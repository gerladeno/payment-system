package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"payment-system/pkg"
	"payment-system/pkg/pgStore"
	"payment-system/pkg/rest"
	"time"
)

type PaymentsHTTPClient struct {
	ConnHTTP *http.Client
	Host     string
}

func (c *PaymentsHTTPClient) CreateWallet(ctx context.Context, wallet string) int {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/createWallet?wallet=%s", c.Host, wallet), nil)
	if err != nil {
		return http.StatusInternalServerError
	}
	resp, err := c.ConnHTTP.Do(req)
	if err != nil {
		return http.StatusInternalServerError
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return resp.StatusCode
}

func (c *PaymentsHTTPClient) GetWallet(ctx context.Context, wallet string) (*pkg.Wallet, string, int) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/getWallet?wallet=%s", c.Host, wallet), nil)
	if err != nil {
		return nil, "", http.StatusInternalServerError
	}
	resp, err := c.ConnHTTP.Do(req)
	if err != nil {
		return nil, "", http.StatusInternalServerError
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	type JSONResponse struct {
		Data  *pkg.Wallet `json:"data,omitempty"`
		Error *string     `json:"message,omitempty"`
		Code  *int        `json:"code,omitempty"`
	}
	var response JSONResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, "", http.StatusInternalServerError
	}
	if resp.StatusCode == http.StatusOK {
		return response.Data, "", http.StatusOK
	}
	return nil, *response.Error, resp.StatusCode
}

func (c *PaymentsHTTPClient) Deposit(ctx context.Context, wallet string, amount float64, key string) int {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/deposit?wallet=%s&amount=%f&key=%s", c.Host, wallet, amount, key), nil)
	if err != nil {
		return http.StatusInternalServerError
	}
	resp, err := c.ConnHTTP.Do(req)
	if err != nil {
		return http.StatusInternalServerError
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return resp.StatusCode
}

func (c *PaymentsHTTPClient) Withdraw(ctx context.Context, wallet string, amount float64, key string) int {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/withdraw?wallet=%s&amount=%f&key=%s", c.Host, wallet, amount, key), nil)
	if err != nil {
		return http.StatusInternalServerError
	}
	resp, err := c.ConnHTTP.Do(req)
	if err != nil {
		return http.StatusInternalServerError
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return resp.StatusCode
}

func (c *PaymentsHTTPClient) TransferFunds(ctx context.Context, from, to string, amount float64, key string) int {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/transferFunds?from=%s&to=%s&amount=%f&key=%s", c.Host, from, to, amount, key), nil)
	if err != nil {
		return http.StatusInternalServerError
	}
	resp, err := c.ConnHTTP.Do(req)
	if err != nil {
		return http.StatusInternalServerError
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return resp.StatusCode
}

func (c *PaymentsHTTPClient) Report(ctx context.Context, wallet string, from, to time.Time, tType int) ([]pgStore.Transaction, string, int) {
	host := fmt.Sprintf("%s/v1/report?wallet=%s&from=%s&to=%s&type=%d", c.Host, wallet, from.Format(rest.DateFmt), to.Format(rest.DateFmt), tType)
	req, err := http.NewRequestWithContext(ctx, "GET", host, nil)
	if err != nil {
		return nil, "", http.StatusInternalServerError
	}
	resp, err := c.ConnHTTP.Do(req)
	if err != nil {
		return nil, "", http.StatusInternalServerError
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	type JSONResponse struct {
		Data  *[]pgStore.Transaction `json:"data,omitempty"`
		Error *string                `json:"message,omitempty"`
		Code  *int                   `json:"code,omitempty"`
	}
	var response JSONResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, "", http.StatusInternalServerError
	}
	if resp.StatusCode == http.StatusOK {
		return *response.Data, "", http.StatusOK
	}
	return nil, *response.Error, resp.StatusCode
}
