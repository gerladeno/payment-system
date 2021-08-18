package pkg

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MetricDBErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "payments",
			Subsystem: "pg_store",
			Name:      "errors",
			Help:      "wallet store err count",
		}, []string{"method"})
	MetricDBTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "payments",
			Subsystem: "pg_store",
			Name:      "db_time",
			Buckets:   []float64{.025, .05, .1, .5, 1, 2.5, 5, 10},
		}, []string{"method"})
)
