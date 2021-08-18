package pgStore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func (pg *PG) Deposit(ctx context.Context, wallet string, amount float64, key string) error {
	return pg.tx(ctx, "Deposit", func(tx *sql.Tx) error {
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
		query = `INSERT INTO transaction (type, wallet, key, amount) VALUES ($1, $2, $3, $4)`
		_, err = tx.ExecContext(ctx, query, transactionDeposit, wallet, key, amount)
		if err != nil {

		}
		return nil
	})
}
func (pg *PG) Withdraw() {
}
func (pg *PG) TransferFunds() {
}
func (pg *PG) Report() {
}

const ownerWalletQuery = `
SELECT wallet
FROM owner
WHERE owner = $1
AND wallet = $2
`

func (pg *PG) CheckOwnerWallet(ctx context.Context, wallet string, owner int) (bool, error) {
	var tmp string
	err := pg.db.GetContext(ctx, &tmp, ownerWalletQuery, owner, wallet)
	switch err {
	case sql.ErrNoRows:
		return false, nil
	case nil:
		return true, nil
	default:
		return false, err
	}
}
