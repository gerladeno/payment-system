package main

import (
	"context"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/onrik/logrus/sentry"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"payment-system/pkg/pgStore"
	"payment-system/pkg/rest"
	"strings"
	"syscall"
	"time"
)

const port = 3000

var version = "0.0.0"

func main() {
	log := getLogger()
	log.Infof("starting payment system service version %s", version)
	ctx := context.Background()
	pg, err := pgStore.GetPGStore(ctx, log, os.Getenv("PG_DSN"))
	if err != nil {
		log.Fatalf("failed to get pgStore: %s", err)
	}
	defer pg.DC()
	if err = pg.Migrate(migrate.Up); err != nil {
		log.Fatalf("err migrating pg store: %s", err)
	}
	router := rest.NewRouter(log, pg, pg, version)
	if err = startServer(ctx, router, log); err != nil {
		log.Fatal(err)
	}
}

func startServer(ctx context.Context, router http.Handler, log *logrus.Logger) error {
	log.Infof("starting server on port %d", port)
	s := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		Handler:           router,
	}
	errCh := make(chan error)
	go func() {
		if err := s.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)
	select {
	case err := <-errCh:
		return err
	case <-sigCh:
	}
	log.Info("terminating...")
	gfCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return s.Shutdown(gfCtx)
}

func getLogger() *logrus.Logger {
	log := logrus.New()
	if strings.ToLower(os.Getenv("VERBOSE")) == "true" {
		log.SetLevel(logrus.DebugLevel)
		log.Debug("log level set to debug")
	}
	log.SetFormatter(&logrus.JSONFormatter{})
	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN == "" {
		return log
	}
	opts := sentry.Options{
		Dsn:              sentryDSN,
		Release:          version,
		AttachStacktrace: true,
		Environment:      os.Getenv("ENV"),
	}
	sentryHook, err := sentry.NewHook(opts, logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel, logrus.WarnLevel)
	if err != nil {
		log.Fatalf("unable to init logger: %s", err)
	}
	log.AddHook(sentryHook)
	log.Info("sentry enabled")
	return log
}
