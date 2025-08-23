package probe

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	successCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prober_success_total",
			Help: "Total successful probe operations",
		},
		[]string{"target_type", "operation_type", "target_host", "target_name"},
	)
	failureCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prober_failure_total",
			Help: "Total failed probe operations",
		},
		[]string{"target_type", "operation_type", "target_host", "target_name"},
	)
)

var probeRegistry = prometheus.NewRegistry()

func InitMetrics() {
	probeRegistry.MustRegister(successCounter)
	probeRegistry.MustRegister(failureCounter)
	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(probeRegistry, promhttp.HandlerOpts{}))
		http.ListenAndServe("127.0.0.1:2112", nil)
	}()
}

func IncProbeSuccess(targetType, opType, targetHost, name string) {
	successCounter.WithLabelValues(targetType, opType, targetHost, name).Inc()
}

func IncProbeFailure(targetType, opType, targetHost, name string) {
	failureCounter.WithLabelValues(targetType, opType, targetHost, name).Inc()
}
