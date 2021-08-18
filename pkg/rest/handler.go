package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"payment-system/pkg"
	"payment-system/pkg/pgStore"
	"regexp"
	"strconv"
)

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
	wallet, err := parseAndValidateWallet(r)
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
	wallet, err := parseAndValidateWallet(r)
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

	err = h.walletStore.Deposit(r.Context(), wallet, amount, key)
}

func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) {
	wallet, err := parseAndValidateWallet(r)
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
}

func (h *Handler) transferFunds(w http.ResponseWriter, r *http.Request) {
}

func (h *Handler) createReport(w http.ResponseWriter, r *http.Request) {
}

func parseAndValidateWallet(r *http.Request) (string, error) {
	uuid := r.URL.Query().Get("wallet")
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
