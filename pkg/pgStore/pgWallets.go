package pgStore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jmoiron/sqlx"
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
const createWalletQuery = `
INSERT INTO wallet (wallet, owner)
VALUES ($1, $2)
ON CONFLICT (wallet) DO NOTHING;
`
const changeBalanceQuery = `
UPDATE wallet SET amount = wallet.amount + $1
WHERE wallet = $2 AND amount >= ($1 * -1)
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

var ErrInsufficientFunds = errors.New("err wallet with uuid specified doesn't have enough money on the balance")
var ErrWalletNotFound = errors.New("err wallet with uuid specified was not found")
var ErrInvalidTransactionType = errors.New("unknown transaction type")

type ErrDuplicateAction string

func (e ErrDuplicateAction) Error() string {
	return fmt.Sprintf("duplicate key: %s", string(e))
}

func (pg *PG) CreateWallet(ctx context.Context, wallet string, owner int) error {
	return pg.tx(ctx, "CreateWallet", func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, createWalletQuery, wallet, owner)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrDuplicateAction(wallet)
		}
		return nil
	})
}

func (pg *PG) DepositWithdraw(ctx context.Context, wallet string, amount float64, key string) error {
	return pg.tx(ctx, "DepositWithdraw", func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, changeBalanceQuery, amount, wallet)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrInsufficientFunds
		}
		var tType TransactionType
		if amount > 0 {
			tType = TransactionDeposit
		} else {
			tType = TransactionWithdrawal
		}
		query := `INSERT INTO transaction (type, wallet, key, amount) VALUES ($1, $2, $3, $4)`
		_, err = tx.ExecContext(ctx, query, tType, wallet, key, amount)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr); pgErr.Code == "23505" {
				return ErrDuplicateAction(key)
			}
			return err
		}
		return nil
	})
}

func (pg *PG) TransferFunds(ctx context.Context, from, to string, amount float64, key string) error {
	return pg.tx(ctx, "TransferFunds", func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, changeBalanceQuery, -amount, from)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrInsufficientFunds
		}
		query := `INSERT INTO transaction (type, wallet, wallet_receiver, key, amount) VALUES ($1, $2, $3, $4, $5)`
		_, err = tx.ExecContext(ctx, query, TransactionTransferFunds, from, to, key, amount)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr); pgErr.Code == "23505" {
				return ErrDuplicateAction(key)
			}
			return err
		}
		_, err = tx.ExecContext(ctx, changeBalanceQuery, amount, to)
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

func (t transaction) tx2Tx() Transaction{
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
	}
	if from != nil {
		queryBuilder.WriteString(fmt.Sprintf("AND ts >= timestamp '%s'\n", from.Format(pgDateTimeFmt)))
	}
	if to != nil {
		queryBuilder.WriteString(fmt.Sprintf("AND ts <= timestamp '%s'\n", to.Format(pgDateTimeFmt)))
	}
	result := make([]Transaction, 0)
	err := pg.tx(ctx, "Report", func(tx *sqlx.Tx) error {
		rows, err := tx.QueryxContext(ctx, queryBuilder.String(), wallet)
		if err != nil {
			return err
		}
		defer func() {
			if err = rows.Close(); err != nil {
				pg.log.Warnf("err closing rows after querying report: %s", err)
			}
		}()
		var tmp transaction
		for rows.Next() {
			err = rows.StructScan(&tmp)
			if err != nil {
				return err
			}
			result = append(result, tmp.tx2Tx())
		}
		return nil
	})
	return result, err
}

func (pg *PG) CheckOwnerWallet(ctx context.Context, wallet string, owner int) (bool, error) {
	var tmp int
	err := pg.db.GetContext(ctx, &tmp, ownerWalletQuery, wallet)
	switch err {
	case sql.ErrNoRows:
		return false, ErrWalletNotFound
	case nil:
		if tmp != owner {
			return false, nil
		}
		return true, nil
	default:
		return false, err
	}
}
