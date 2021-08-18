package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gocarina/gocsv"
	"github.com/sirupsen/logrus"
	"net/http"
	"payment-system/pkg"
	"payment-system/pkg/pgStore"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const dateFmt = `2006-01-02`

var ErrInvalidUUIDFormat = errors.New("err invalid uuid format")
var ErrWalletNotSpecified = errors.New("err wallet not specified in the query")
var uuidReqexp = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[89aAbB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")

type JSONResponse struct {
	Data  *interface{} `json:"data,omitempty"`
	Error *string      `json:"message,omitempty"`
	Code  *int         `json:"code,omitempty"`
}

type Handler struct {
	walletStore WalletStore
	log         *logrus.Logger
}

func NewHandler(log *logrus.Logger, walletStore WalletStore) *Handler {
	return &Handler{
		walletStore: walletStore,
		log:         log,
	}
}

func (h *Handler) createWallet(w http.ResponseWriter, r *http.Request) {
	wallet, err := parseAndValidateWallet(r, "wallet")
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	owner := pkg.ClientFromCtx(r.Context()).ID
	if err = h.walletStore.CreateWallet(r.Context(), wallet, owner); err != nil {
		if _, ok := err.(pgStore.ErrDuplicateAction); ok {
			writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
			return
		}
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}
	writeOkResponse(w, "ok")
}

func (h *Handler) deposit(w http.ResponseWriter, r *http.Request) {
	wallet, err := parseAndValidateWallet(r, "wallet")
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	amount, err := parseAmount(r)
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	if amount <= 0 {
		writeErrResponse(w, "Bad Request: can't deposit negative amount", http.StatusBadRequest)
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		writeErrResponse(w, "Bad Request: transaction key not specified", http.StatusBadRequest)
		return
	}
	_, err = h.walletStore.CheckOwnerWallet(r.Context(), wallet, 0)
	switch err {
	case pgStore.ErrWalletNotFound:
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	case nil:
	default:
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}
	err = h.walletStore.DepositWithdraw(r.Context(), wallet, amount, key)
	switch err {
	case pgStore.ErrInsufficientFunds:
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	case nil:
	default:
		if _, ok := err.(pgStore.ErrDuplicateAction); ok {
			writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
			return
		}
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}
	writeOkResponse(w, "ok")
}

func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) {
	wallet, err := parseAndValidateWallet(r, "wallet")
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	amount, err := parseAmount(r)
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	if amount <= 0 {
		writeErrResponse(w, "Bad Request: specify positive amount to withdraw", http.StatusBadRequest)
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		writeErrResponse(w, "Bad Request: transaction key not specified", http.StatusBadRequest)
		return
	}
	owner := pkg.ClientFromCtx(r.Context()).ID
	ok, err := h.walletStore.CheckOwnerWallet(r.Context(), wallet, owner)
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}
	if !ok {
		writeErrResponse(w, "Forbidden", http.StatusForbidden)
		return
	}
	err = h.walletStore.DepositWithdraw(r.Context(), wallet, -amount, key)
	switch err {
	case pgStore.ErrInsufficientFunds:
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	case nil:
	default:
		if _, ok := err.(pgStore.ErrDuplicateAction); ok {
			writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
			return
		}
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}
	writeOkResponse(w, "ok")
}

func (h *Handler) transferFunds(w http.ResponseWriter, r *http.Request) {
	from, err := parseAndValidateWallet(r, "from")
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	to, err := parseAndValidateWallet(r, "to")
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	amount, err := parseAmount(r)
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	if amount <= 0 {
		writeErrResponse(w, "Bad Request: specify positive amount to transfer", http.StatusBadRequest)
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		writeErrResponse(w, "Bad Request: transaction key not specified", http.StatusBadRequest)
		return
	}
	owner := pkg.ClientFromCtx(r.Context()).ID
	ok, err := h.walletStore.CheckOwnerWallet(r.Context(), from, owner)
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}
	if !ok {
		writeErrResponse(w, "Forbidden", http.StatusForbidden)
		return
	}
	_, err = h.walletStore.CheckOwnerWallet(r.Context(), to, 0)
	switch err {
	case pgStore.ErrWalletNotFound:
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	case nil:
	default:
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}
	err = h.walletStore.TransferFunds(r.Context(), from, to, amount, key)
	switch err {
	case pgStore.ErrInsufficientFunds:
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	case nil:
	default:
		if _, ok := err.(pgStore.ErrDuplicateAction); ok {
			writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
			return
		}
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}
	writeOkResponse(w, "ok")
}

func (h *Handler) createReport(w http.ResponseWriter, r *http.Request) {
	wallet, err := parseAndValidateWallet(r, "wallet")
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	from, err := parseDate(r, "from")
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	to, err := parseDate(r, "to")
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	tType, err := parseTransactionType(r)
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
		return
	}
	owner := pkg.ClientFromCtx(r.Context()).ID
	ok, err := h.walletStore.CheckOwnerWallet(r.Context(), wallet, owner)
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}
	if !ok {
		writeErrResponse(w, "Forbidden", http.StatusForbidden)
		return
	}
	transactions, err := h.walletStore.Report(r.Context(), wallet, from, to, tType)
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}
	if csv := r.URL.Query().Get("csv"); csv == "" {
		writeOkResponse(w, transactions)
		return
	}
	w.Header().Set("Content-type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=Report.csv")
	data, err := toCsv(transactions)
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(data)
	if err != nil {
		writeErrResponse(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
	}
}

func parseTransactionType(r *http.Request) (pgStore.TransactionType, error) {
	s := r.URL.Query().Get("type")
	switch strings.ToLower(s) {
	case "0", "deposit":
		return pgStore.TransactionDeposit, nil
	case "1", "withdrawal", "withdraw":
		return pgStore.TransactionWithdrawal, nil
	case "2", "transfer", "transferfrom":
		return pgStore.TransactionTransferFunds, nil
	case "3", "transferto":
		return pgStore.TransactionTransferFundsTo, nil
	case "":
		return pgStore.AllTransactions, nil
	default:
		return 0, pgStore.ErrInvalidTransactionType
	}
}

func parseDate(r *http.Request, name string) (*time.Time, error) {
	s := r.URL.Query().Get(name)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(dateFmt, s)
	return &t, err
}

func parseAndValidateWallet(r *http.Request, name string) (string, error) {
	uuid := r.URL.Query().Get(name)
	if uuid == "" {
		return "", ErrWalletNotSpecified
	}
	if !isValidUUID(uuid) {
		return "", ErrInvalidUUIDFormat
	}
	return uuid, nil
}

func isValidUUID(uuid string) bool {
	return uuidReqexp.MatchString(uuid)
}

func parseAmount(r *http.Request) (float64, error) {
	value := r.URL.Query().Get("amount")
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return amount, nil
}

func writeErrResponse(w http.ResponseWriter, err string, status int) {
	w.WriteHeader(status)
	w.Header().Set("Content-type", "application/json")
	response := JSONResponse{
		Error: &err,
		Code:  &status,
	}
	_ = json.NewEncoder(w).Encode(response)
}

func writeOkResponse(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-type", "application/json")
	ok := http.StatusOK
	_ = json.NewEncoder(w).Encode(JSONResponse{Data: &data, Code: &ok})
}

func toCsv(data interface{}) ([]byte, error) {
	result, err := gocsv.MarshalBytes(data)
	if err != nil {
		return nil, err
	}
	return result, nil
}
