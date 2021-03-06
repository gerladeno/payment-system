package rest

import (
	"compress/flate"
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"net/http"
	"payment-system/pkg"
	"payment-system/pkg/pgStore"
	"time"
)

type ClientStore interface {
}

type WalletStore interface {
	GetWallet(ctx context.Context, wallet string) (pkg.Wallet, error)
	CreateWallet(ctx context.Context, wallet string, owner int) error
	DepositWithdraw(ctx context.Context, wallet string, amount float64, key string) error
	TransferFunds(ctx context.Context, from, to string, amount float64, key string) error
	Report(ctx context.Context, wallet string, from, to *time.Time, tType pgStore.TransactionType) ([]pgStore.Transaction, error)
	CheckOwnerWallet(ctx context.Context, wallet string, owner int) (bool, error)
}

func NewRouter(log *logrus.Logger, clientStore ClientStore, walletStore WalletStore, version string) *chi.Mux {
	r := chi.NewRouter()
	h := NewHandler(log, walletStore)
	r.Use(middleware.Recoverer)
	r.Use(cors.AllowAll().Handler)
	r.Use(middleware.NewCompressor(flate.DefaultCompression).Handler)
	r.NotFound(notFoundHandler)
	r.Get("/ping", pingHandler)
	r.Get("/version", versionHandler(version))
	r.Get("/metrics", promhttp.Handler().ServeHTTP)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log, NoColor: true}))
		r.Use(middleware.Timeout(30 * time.Second))
		r.Use(middleware.Throttle(30))
		r.Use(httprate.LimitByIP(1000, time.Minute))
		r.Use(auth(clientStore))
		r.Route("/v1", func(r chi.Router) {
			r.Get("/createWallet", h.CreateWallet)
			r.Get("/getWallet", h.GetWallet)
			r.Get("/deposit", h.Deposit)
			r.Get("/withdraw", h.Withdraw)
			r.Get("/transferFunds", h.TransferFunds)
			r.Get("/report", h.CreateReport)
		})
	})
	return r
}

func notFoundHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "404 page not found. Check docs: https://github.com/gerladeno/payment-system", http.StatusNotFound)
}

func pingHandler(w http.ResponseWriter, _ *http.Request) {
	if _, err := w.Write([]byte("pong")); err != nil {
		http.Error(w, "pong error", http.StatusInternalServerError)
	}
}

func versionHandler(version string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(version))
	}
}
