package pgStore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"payment-system/pkg"
	"strings"
	"time"
)

type TransactionType int8

const (
	TransactionDeposit TransactionType = iota
	TransactionWithdrawal
	TransactionTransferFunds
	TransactionTransferFundsTo
	AllTransactions = -1
)
const pgDateTimeFmt = `2006-01-02 15:04:05`
const getWalletQuery = `
SELECT wallet, amount, owner, status, updated, created
FROM wallet
WHERE wallet = $1
`
const createWalletQuery = `
INSERT INTO wallet (wallet, owner)
VALUES ($1, $2)
ON CONFLICT (wallet) DO NOTHING;
`
const changeBalanceQuery = `
UPDATE wallet SET amount = wallet.amount + $1::numeric(12, 2)
WHERE wallet = $2 AND amount >= ($1::numeric(12, 2) * -1)
`
const walletReportTmpl = `
SELECT id, type, wallet, wallet_receiver, key, amount, ts
FROM transaction
WHERE 1=1
`
const ownerWalletQuery = `
SELECT owner
FROM wallet
WHERE wallet = $1
`

func (pg *PG) GetWallet(ctx context.Context, wallet string) (pkg.Wallet, error) {
	result := pkg.Wallet{}
	err := pg.tx(ctx, "GetWallet", func(tx pgx.Tx) error {
		return pgxscan.Get(ctx, tx, &result, getWalletQuery, wallet)
	})
	return result, err
}

func (pg *PG) CreateWallet(ctx context.Context, wallet string, owner int) error {
	return pg.tx(ctx, "CreateWallet", func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, createWalletQuery, wallet, owner)
		if err != nil {
			return err
		}
		n := result.RowsAffected()
		if n == 0 {
			return pkg.ErrDuplicateAction(wallet)
		}
		return nil
	})
}

func (pg *PG) DepositWithdraw(ctx context.Context, wallet string, amount float64, key string) error {
	return pg.tx(ctx, "DepositWithdraw", func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, changeBalanceQuery, amount, wallet)
		if err != nil {
			return err
		}
		n := result.RowsAffected()
		if n == 0 {
			return pkg.ErrInsufficientFunds
		}
		var tType TransactionType
		if amount > 0 {
			tType = TransactionDeposit
		} else {
			tType = TransactionWithdrawal
		}
		query := `INSERT INTO transaction (type, wallet, key, amount) VALUES ($1, $2, $3, $4)`
		_, err = tx.Exec(ctx, query, tType, wallet, key, amount)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr); pgErr.Code == "23505" {
				return pkg.ErrDuplicateAction(key)
			}
			return err
		}
		return nil
	})
}

func (pg *PG) TransferFunds(ctx context.Context, from, to string, amount float64, key string) error {
	return pg.tx(ctx, "TransferFunds", func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, changeBalanceQuery, -amount, from)
		if err != nil {
			return err
		}
		n := result.RowsAffected()
		if n == 0 {
			return pkg.ErrInsufficientFunds
		}
		query := `INSERT INTO transaction (type, wallet, wallet_receiver, key, amount) VALUES ($1, $2, $3, $4, $5)`
		_, err = tx.Exec(ctx, query, TransactionTransferFunds, from, to, key, amount)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr); pgErr.Code == "23505" {
				return pkg.ErrDuplicateAction(key)
			}
			return err
		}
		_, err = tx.Exec(ctx, changeBalanceQuery, amount, to)
		if err != nil {
			return err
		}
		return nil
	})
}

type Transaction struct {
	ID             int64           `json:"id" csv:"ID"`
	Type           TransactionType `json:"type" csv:"TYPE"`
	Wallet         string          `json:"wallet" csv:"WALLET"`
	WalletReceiver string          `json:"wallet_receiver" csv:"WALLET_RECEIVER"`
	Key            string          `json:"key" csv:"KEY"`
	Amount         float64         `json:"amount" csv:"AMOUNT"`
	Ts             time.Time       `json:"ts" csv:"TS"`
}

type transaction struct {
	ID             int64           `db:"id"`
	Type           TransactionType `db:"type"`
	Wallet         string          `db:"wallet"`
	WalletReceiver sql.NullString  `db:"wallet_receiver"`
	Key            string          `db:"key"`
	Amount         float64         `db:"amount"`
	Ts             time.Time       `db:"ts"`
}

func (t transaction) tx2Tx() Transaction {
	return Transaction{
		ID:             t.ID,
		Type:           t.Type,
		Wallet:         t.Wallet,
		WalletReceiver: t.WalletReceiver.String,
		Key:            t.Key,
		Amount:         t.Amount,
		Ts:             t.Ts,
	}
}

func (pg *PG) Report(ctx context.Context, wallet string, from, to *time.Time, tType TransactionType) ([]Transaction, error) {
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(walletReportTmpl)
	switch tType {
	case TransactionTransferFundsTo:
		queryBuilder.WriteString("AND wallet_receiver = $1\n")
	case TransactionDeposit:
		queryBuilder.WriteString("AND wallet = $1 AND type = 0\n")
	case TransactionWithdrawal:
		queryBuilder.WriteString("AND wallet = $1 AND type = 1\n")
	case TransactionTransferFunds:
		queryBuilder.WriteString("AND wallet = $1 AND type = 2\n")
	case AllTransactions:
		queryBuilder.WriteString("AND (wallet = $1 OR wallet_receiver = $1)\n")
	default:
		return nil, pkg.ErrInvalidTransactionType
	}
	if from != nil {
		queryBuilder.WriteString(fmt.Sprintf("AND ts >= timestamp '%s'\n", from.Format(pgDateTimeFmt)))
	}
	if to != nil {
		queryBuilder.WriteString(fmt.Sprintf("AND ts <= timestamp '%s'\n", to.Format(pgDateTimeFmt)))
	}
	result := make([]Transaction, 0)
	tmp := make([]transaction, 0)
	err := pg.tx(ctx, "Report", func(tx pgx.Tx) error {
		err := pgxscan.Select(ctx, tx, &tmp, queryBuilder.String(), wallet)
		if err != nil {
			return err
		}
		for _, tr := range tmp {
			result = append(result, tr.tx2Tx())
		}
		return nil
	})
	return result, err
}

func (pg *PG) CheckOwnerWallet(ctx context.Context, wallet string, owner int) (bool, error) {
	var tmp int
	err := pg.tx(ctx, "CheckOwnerWallet", func(tx pgx.Tx) error {
		return pgxscan.Get(ctx, tx, &tmp, ownerWalletQuery, wallet)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, pkg.ErrWalletNotFound
		}
		return false, err
	}
	if tmp != owner {
		return false, nil
	}
	return true, nil
}
