package observability

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Explicit registration (instead of promauto.NewCounterVec) so the
// metrics are only registered with the default registry exactly once
// per process. The previous promauto-based design re-registered on
// every package init, which panics with
//   "duplicate metrics collector registration attempted"
// when the package is loaded more than once in the same process —
// a real risk under `go test -count=N` and integration suites that
// re-import the same package across test binaries.
//
// init() registers the vectors at process start so the metrics are
// visible to any Prometheus scrape endpoint that walks the default
// registry, satisfying the README's monitoring promise. The
// registerOnce keeps the registration idempotent — even if init()
// somehow runs more than once (e.g. a future refactor splits the
// init into multiple steps, or a test framework re-execs init),
// prometheus.MustRegister is only called on the first run, so the
// duplicate-registration panic is impossible.

var (
	ExportsTotal    *prometheus.CounterVec
	ExportDuration  *prometheus.HistogramVec
	metricsInitOnce = make(chan struct{})
	registerOnce    sync.Once
)

func init() {
	ExportsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "vectorx_trace_exports_total",
		Help: "Total trace export calls per exporter and status",
	}, []string{"exporter", "status"})

	ExportDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "vectorx_trace_export_duration_seconds",
		Help:    "Trace export call duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"exporter"})

	registerOnce.Do(func() {
		prometheus.MustRegister(ExportsTotal, ExportDuration)
	})
	close(metricsInitOnce)
}

// ObserveExport is a convenience for adapter code that wants to record
// an export call without directly poking the *Vec types. Safe to call
// from any goroutine.
func ObserveExport(exporter, status string, durationSeconds float64) {
	<-metricsInitOnce
	ExportsTotal.WithLabelValues(exporter, status).Inc()
	ExportDuration.WithLabelValues(exporter).Observe(durationSeconds)
}
