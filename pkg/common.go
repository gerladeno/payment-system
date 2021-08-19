package pkg

import (
	"errors"
	"fmt"
	"time"
)

var ErrInsufficientFunds = errors.New("err wallet with uuid specified doesn't have enough money on the balance")
var ErrWalletNotFound = errors.New("err wallet with uuid specified was not found")
var ErrInvalidTransactionType = errors.New("unknown transaction type")

type ErrDuplicateAction string

func (e ErrDuplicateAction) Error() string {
	return fmt.Sprintf("duplicate key: %s", string(e))
}

type Wallet struct {
	Amount  float64   `db:"amount" json:"amount"`
	Wallet  string    `db:"wallet" json:"wallet"`
	Owner   int       `db:"owner" json:"owner"`
	Status  int8      `db:"status" json:"status"`
	Updated time.Time `db:"updated" json:"updated"`
	Created time.Time `db:"created" json:"created"`
}
