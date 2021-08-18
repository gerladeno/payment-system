package pgStore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
)

type transactionType int8

const (
	transactionDeposit transactionType = iota
	transactionWithdrawal
	transactionTransferFunds
)

var ErrInsufficientFunds = errors.New("err wallet with uuid specified doesn't have enough money on the balance")
var ErrWalletNotFound = errors.New("err wallet with uuid specified was not found")

type ErrDuplicateAction string

func (e ErrDuplicateAction) Error() string {
	return fmt.Sprintf("duplicate key: %s", string(e))
}

const createWalletQuery = `
INSERT INTO wallet (wallet, owner)
VALUES ($1, $2)
ON CONFLICT (wallet) DO NOTHING;
`

func (pg *PG) CreateWallet(ctx context.Context, wallet string, owner int) error {
	return pg.tx(ctx, "CreateWallet", func(tx *sql.Tx) error {
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
	return pg.tx(ctx, "DepositWithdraw", func(tx *sql.Tx) error {
		query := `
UPDATE wallet SET amount = wallet.amount + $1
WHERE wallet = $2 AND amount >= ($1 * -1)`
		result, err := tx.ExecContext(ctx, query, amount, wallet)
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
		var tType transactionType
		if amount > 0 {
			tType = transactionDeposit
		} else {
			tType = transactionWithdrawal
		}
		query = `INSERT INTO transaction (type, wallet, key, amount) VALUES ($1, $2, $3, $4)`
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
	return pg.tx(ctx, "DepositWithdraw", func(tx *sql.Tx) error {
		query := `
UPDATE wallet SET amount = wallet.amount + $1
WHERE wallet = $2 AND amount >= ($1 * -1)`
		result, err := tx.ExecContext(ctx, query, -amount, from)
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
		query = `INSERT INTO transaction (type, wallet, wallet_receiver, key, amount) VALUES ($1, $2, $3, $4, $5)`
		_, err = tx.ExecContext(ctx, query, transactionTransferFunds, from, to, key, amount)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr); pgErr.Code == "23505" {
				return ErrDuplicateAction(key)
			}
			return err
		}
		query = `
UPDATE wallet SET amount = wallet.amount + $1
WHERE wallet = $2 AND amount >= ($1 * -1)`
		result, err = tx.ExecContext(ctx, query, amount, to)
		if err != nil {
			return err
		}
		return nil
	})
}
func (pg *PG) Report() {
}

const ownerWalletQuery = `
SELECT owner
FROM wallet
WHERE wallet = $1
`

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
