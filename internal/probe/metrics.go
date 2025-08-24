package probe

import (
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	successCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prober_success_total",
			Help: "Total successful probe operations",
		},
		[]string{"target_type", "operation_type", "target_name", "source_region", "destination_region", "source_node_name", "source_node_ip"},
	)
	failureCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prober_failure_total",
			Help: "Total failed probe operations",
		},
		[]string{"target_type", "operation_type", "target_name", "source_region", "destination_region", "source_node_name", "source_node_ip"},
	)
)

var SourceNodeName string
var SourceNodeIP string

var probeRegistry = prometheus.NewRegistry()

func InitMetrics() {
	// Set source node info once from env
	SourceNodeName = os.Getenv("NODE_NAME")
	if SourceNodeName == "" {
		SourceNodeName = "local"
	}
	SourceNodeIP = os.Getenv("NODE_IP")
	if SourceNodeIP == "" {
		SourceNodeIP = "unknown"
	}
	probeRegistry.MustRegister(successCounter)
	probeRegistry.MustRegister(failureCounter)
	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(probeRegistry, promhttp.HandlerOpts{}))
		http.ListenAndServe("127.0.0.1:2112", nil)
	}()
}

func IncProbeSuccess(targetType, opType, name, sourceRegion, destinationRegion string) {
	successCounter.WithLabelValues(targetType, opType, name, sourceRegion, destinationRegion, SourceNodeName, SourceNodeIP).Inc()
}

func IncProbeFailure(targetType, opType, name, sourceRegion, destinationRegion string) {
	failureCounter.WithLabelValues(targetType, opType, name, sourceRegion, destinationRegion, SourceNodeName, SourceNodeIP).Inc()
}
