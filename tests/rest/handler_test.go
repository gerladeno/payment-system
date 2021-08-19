package rest_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"io"
	"net/http"
	"net/http/httptest"
	"payment-system/pkg"
	"payment-system/pkg/pgStore"
	"payment-system/pkg/rest"
	"sync"
	"testing"
	"time"
)

// TODO update tests after auth

type RESTSuite struct {
	h *rest.Handler
	suite.Suite
}

func (s *RESTSuite) SetupSuite() {
	log := &logrus.Logger{}
	fs := FakeStore{}
	s.h = rest.NewHandler(log, fs)
}

func (s *RESTSuite) TestGetWallet() {
	host := "/getWallet?wallet=rubbish"
	code, _ := s.processGetWithHandler(host, s.h.GetWallet)
	require.Equal(s.T(), code, http.StatusBadRequest)
	host = "/getWallet?wallet="
	code, _ = s.processGetWithHandler(host, s.h.GetWallet)
	require.Equal(s.T(), code, http.StatusBadRequest)
	host = "/getWallet"
	code, _ = s.processGetWithHandler(host, s.h.GetWallet)
	require.Equal(s.T(), code, http.StatusBadRequest)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			host := fmt.Sprintf("/getWallet?wallet=%s", uuid.New().String())
			code, _ := s.processGetWithHandler(host, s.h.GetWallet)
			require.Equal(s.T(), code, http.StatusOK)
		}()
	}
	wg.Wait()
}

func (s *RESTSuite) TestCreateWallet() {
	host := "/createWallet?wallet=rubbish"
	code, _ := s.processGetWithHandler(host, s.h.CreateWallet)
	require.Equal(s.T(), code, http.StatusBadRequest)
	host = "/createWallet?wallet="
	code, _ = s.processGetWithHandler(host, s.h.CreateWallet)
	require.Equal(s.T(), code, http.StatusBadRequest)
	host = "/createWallet"
	code, _ = s.processGetWithHandler(host, s.h.CreateWallet)
	require.Equal(s.T(), code, http.StatusBadRequest)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			host := fmt.Sprintf("/createWallet?wallet=%s", uuid.New().String())
			code, _ := s.processGetWithHandler(host, s.h.CreateWallet)
			require.Equal(s.T(), code, http.StatusOK)
		}()
	}
	wg.Wait()
}

func (s *RESTSuite) TestDeposit() {
	host := fmt.Sprintf("/deposit?wallet=%s&key=a", uuid.New().String())
	code, _ := s.processGetWithHandler(host, s.h.Deposit)
	require.Equal(s.T(), code, http.StatusBadRequest)
	code, _ = s.processGetWithHandler(host+"&amount=-10", s.h.Deposit)
	require.Equal(s.T(), code, http.StatusBadRequest)
	code, _ = s.processGetWithHandler(host+"&amount=0", s.h.Deposit)
	require.Equal(s.T(), code, http.StatusBadRequest)
	code, _ = s.processGetWithHandler(host+"&amount=asda", s.h.Deposit)
	require.Equal(s.T(), code, http.StatusBadRequest)
	code, _ = s.processGetWithHandler(fmt.Sprintf("/deposit?wallet=%s", uuid.New().String())+"&amount=10", s.h.Deposit)
	require.Equal(s.T(), code, http.StatusBadRequest)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			host := fmt.Sprintf("/deposit?wallet=%s&key=a", uuid.New().String())
			code, _ := s.processGetWithHandler(host+"&amount=10", s.h.Deposit)
			require.Equal(s.T(), code, http.StatusOK)
		}()
	}
	wg.Wait()
}

func (s *RESTSuite) TestWithdraw() {
	host := fmt.Sprintf("/withdraw?wallet=%s&key=a", uuid.New().String())
	code, _ := s.processGetWithHandler(host, s.h.Withdraw)
	require.Equal(s.T(), code, http.StatusBadRequest)
	code, _ = s.processGetWithHandler(host+"&amount=-10", s.h.Withdraw)
	require.Equal(s.T(), code, http.StatusBadRequest)
	code, _ = s.processGetWithHandler(host+"&amount=0", s.h.Withdraw)
	require.Equal(s.T(), code, http.StatusBadRequest)
	code, _ = s.processGetWithHandler(host+"&amount=asda", s.h.Withdraw)
	require.Equal(s.T(), code, http.StatusBadRequest)
	code, _ = s.processGetWithHandler(fmt.Sprintf("/withdraw?wallet=%s", uuid.New().String())+"&amount=10", s.h.Withdraw)
	require.Equal(s.T(), code, http.StatusBadRequest)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			host := fmt.Sprintf("/deposit?wallet=%s&key=a", uuid.New().String())
			code, _ := s.processGetWithHandler(host+"&amount=10", s.h.Withdraw)
			require.Equal(s.T(), code, http.StatusOK)
		}()
	}
	wg.Wait()
}

func (s *RESTSuite) TestTransferFunds() {
	host := fmt.Sprintf("/transferFunds?from=%s&to=%s&key=a&amount=-100", uuid.New().String(), uuid.New().String())
	code, _ := s.processGetWithHandler(host, s.h.TransferFunds)
	require.Equal(s.T(), code, http.StatusBadRequest)
	host = fmt.Sprintf("/transferFunds?from=%s&key=a&amount=100", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.TransferFunds)
	require.Equal(s.T(), code, http.StatusBadRequest)
	host = fmt.Sprintf("/transferFunds?to=%s&key=a&amount=100", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.TransferFunds)
	require.Equal(s.T(), code, http.StatusBadRequest)
	host = fmt.Sprintf("/transferFunds?from=%s&to=%s&key=a", uuid.New().String(), uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.TransferFunds)
	require.Equal(s.T(), code, http.StatusBadRequest)
	host = fmt.Sprintf("/transferFunds?from=%s&to=%s&amount=100", uuid.New().String(), uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.TransferFunds)
	require.Equal(s.T(), code, http.StatusBadRequest)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			host := fmt.Sprintf("/transferFunds?from=%s&to=%s&key=a&amount=100", uuid.New().String(), uuid.New().String())
			code, _ := s.processGetWithHandler(host, s.h.TransferFunds)
			require.Equal(s.T(), code, http.StatusOK)
		}()
	}
	wg.Wait()
}

func (s *RESTSuite) TestReport() {
	host := fmt.Sprintf("/report?wallet=%s&from=2021-06-01&to=%s&type=0", uuid.New().String(), time.Now().Format("2006-01-02"))
	code, _ := s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s&from=2021-06-01&to=%s&type=deposit", uuid.New().String(), time.Now().Format("2006-01-02"))
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s&from=2021-06-01&type=1", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s&from=2021-06-01&type=withdraw", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s&from=2021-06-01&type=withdrawal", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s&to=2021-06-01&type=2", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s&to=2021-06-01&type=transfer", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s&to=2021-06-01&type=transferfrom", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s&to=2021-06-01&type=3", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s&to=2021-06-01&type=transferto", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s&type=0", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusOK)
	host = fmt.Sprintf("/report?wallet=%s&type=4", uuid.New().String())
	code, _ = s.processGetWithHandler(host, s.h.CreateReport)
	require.Equal(s.T(), code, http.StatusBadRequest)
}

func (s *RESTSuite) processGetWithHandler(host string, handler func(w http.ResponseWriter, r *http.Request)) (code int, body []byte) {
	req, err := http.NewRequest("GET", host, nil)
	require.NoError(s.T(), err)
	w := httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	body, err = io.ReadAll(resp.Body)
	require.NoError(s.T(), err)
	return resp.StatusCode, body
}

func TestRESTSuite(t *testing.T) {
	suite.Run(t, new(RESTSuite))
}

type FakeStore struct {
}

func (f FakeStore) GetWallet(_ context.Context, _ string) (pkg.Wallet, error) {
	return pkg.Wallet{}, nil
}
func (f FakeStore) CreateWallet(_ context.Context, _ string, _ int) error {
	return nil
}
func (f FakeStore) DepositWithdraw(_ context.Context, _ string, _ float64, _ string) error {
	return nil
}
func (f FakeStore) TransferFunds(_ context.Context, _, _ string, _ float64, _ string) error {
	return nil
}
func (f FakeStore) Report(_ context.Context, _ string, _, _ *time.Time, _ pgStore.TransactionType) ([]pgStore.Transaction, error) {
	return make([]pgStore.Transaction, 0), nil
}
func (f FakeStore) CheckOwnerWallet(_ context.Context, _ string, _ int) (bool, error) {
	return true, nil
}
