// build +integration

package integration_test

import (
	"context"
	"crypto/tls"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"net/http"
	"payment-system/tests/integration"
	"testing"
	"time"
)

type PaymentsSuite struct {
	suite.Suite
	client *integration.PaymentsHTTPClient
}

func (s *PaymentsSuite) SetupSuite() {
	connHTTP := http.Client{}
	connHTTP.Transport = http.DefaultTransport
	connHTTP.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	s.client = &integration.PaymentsHTTPClient{ConnHTTP: &connHTTP, Host: "http://0.0.0.0:3000"}
}

func TestPaymentSuite(t *testing.T) {
	suite.Run(t, new(PaymentsSuite))
}

func (s *PaymentsSuite) TestScenario() {
	// run ONLY on empty DB
	//s.T().Skip()
	ctx := context.Background()
	uid1 := uuid.New().String()
	uid2 := uuid.New().String()
	uid3 := uuid.New().String()
	// creating
	code := s.client.CreateWallet(ctx, uid1)
	require.Equal(s.T(), code, http.StatusOK)
	code = s.client.CreateWallet(ctx, uid1)
	require.Equal(s.T(), code, http.StatusBadRequest)
	code = s.client.CreateWallet(ctx, uid2)
	require.Equal(s.T(), code, http.StatusOK)
	// checking
	wallet, _, code := s.client.GetWallet(ctx, uid1)
	require.Equal(s.T(), code, http.StatusOK)
	require.Equal(s.T(), wallet.Wallet, uid1)
	wallet, _, code = s.client.GetWallet(ctx, uid2)
	require.Equal(s.T(), code, http.StatusOK)
	require.Equal(s.T(), wallet.Wallet, uid2)
	wallet, text, code := s.client.GetWallet(ctx, uid3)
	require.Equal(s.T(), code, http.StatusNotFound)
	require.Nil(s.T(), wallet)
	require.Equal(s.T(), text, "Not Found: err wallet with uuid specified was not found")
	// depositing
	code = s.client.Deposit(ctx, uid1, 1000.57, "12")
	require.Equal(s.T(), code, http.StatusOK)
	code = s.client.Deposit(ctx, uid1, 1000.57, "12")
	require.Equal(s.T(), code, http.StatusBadRequest)
	wallet, _, code = s.client.GetWallet(ctx, uid1)
	require.Equal(s.T(), code, http.StatusOK)
	require.Equal(s.T(), wallet.Amount, 1000.57)
	// withdrawing
	code = s.client.Withdraw(ctx, uid1, 20.1, "13")
	require.Equal(s.T(), code, http.StatusOK)
	code = s.client.Withdraw(ctx, uid1, 20.1, "14")
	require.Equal(s.T(), code, http.StatusOK)
	code = s.client.Withdraw(ctx, uid1, 20.1, "15")
	require.Equal(s.T(), code, http.StatusOK)
	code = s.client.Withdraw(ctx, uid1, 20.1, "16")
	require.Equal(s.T(), code, http.StatusOK)
	wallet, _, code = s.client.GetWallet(ctx, uid1)
	require.Equal(s.T(), code, http.StatusOK)
	require.Equal(s.T(), wallet.Amount, 920.17)
	// transferring
	code = s.client.TransferFunds(ctx, uid1, uid2, 40, "17")
	require.Equal(s.T(), code, http.StatusOK)
	code = s.client.TransferFunds(ctx, uid1, uid2, 40, "18")
	require.Equal(s.T(), code, http.StatusOK)
	code = s.client.TransferFunds(ctx, uid1, uid2, 40, "19")
	require.Equal(s.T(), code, http.StatusOK)
	wallet, _, code = s.client.GetWallet(ctx, uid1)
	require.Equal(s.T(), code, http.StatusOK)
	require.Equal(s.T(), wallet.Amount, 800.17)
	wallet, _, code = s.client.GetWallet(ctx, uid2)
	require.Equal(s.T(), code, http.StatusOK)
	require.Equal(s.T(), wallet.Amount, 120.0)
	// reports
	txs, _, code := s.client.Report(ctx, uid1, time.Time{}, time.Now(), -1)
	require.Len(s.T(), txs, 8)
	require.Equal(s.T(), code, http.StatusOK)
	txs, _, code = s.client.Report(ctx, uid1, time.Time{}, time.Now(), 0)
	require.Len(s.T(), txs, 1)
	require.Equal(s.T(), code, http.StatusOK)
	txs, _, code = s.client.Report(ctx, uid1, time.Time{}, time.Now(), 1)
	require.Len(s.T(), txs, 4)
	require.Equal(s.T(), code, http.StatusOK)
	txs, _, code = s.client.Report(ctx, uid1, time.Time{}, time.Now(), 2)
	require.Len(s.T(), txs, 3)
	require.Equal(s.T(), code, http.StatusOK)
	txs, _, code = s.client.Report(ctx, uid1, time.Time{}, time.Now(), 3)
	require.Len(s.T(), txs, 0)
	require.Equal(s.T(), code, http.StatusOK)
	txs, _, code = s.client.Report(ctx, uid2, time.Time{}, time.Now(), 3)
	require.Len(s.T(), txs, 3)
	require.Equal(s.T(), code, http.StatusOK)
	txs, _, code = s.client.Report(ctx, uid2, time.Time{}, time.Now(), -1)
	require.Len(s.T(), txs, 3)
	require.Equal(s.T(), code, http.StatusOK)
}