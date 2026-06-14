package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ExportsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vectorx_trace_exports_total",
		Help: "Total trace export calls per exporter and status",
	}, []string{"exporter", "status"})

	ExportDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "vectorx_trace_export_duration_seconds",
		Help:    "Trace export call duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"exporter"})
)
