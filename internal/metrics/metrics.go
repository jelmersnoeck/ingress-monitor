package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ingressMonitorTotalGauge   = "ingressmonitor_ingressmonitor_total"
	ingressMonitorSyncGauge    = "ingressmonitor_ingressmonitor_sync_total"
	ingressMonitorFailedGauge  = "ingressmonitor_ingressmonitor_failed_total"
	ingressMonitorSuccessGauge = "ingressmonitor_ingressmonitor_success_total"
)

// Namespaced represent a type which has a namespace attached to it.
type Namespaced interface {
	Namespace() string
}

// Metrics is a wrapper for the metrics we use within the operator.
type Metrics struct {
	ingressMonitorTotalGauge   *prometheus.GaugeVec
	ingressMonitorSyncGauge    *prometheus.GaugeVec
	ingressMonitorFailedGauge  *prometheus.GaugeVec
	ingressMonitorSuccessGauge *prometheus.GaugeVec
}

// IngressMonitorMetric represents a metric which will be used to capture
// information about an IngressMonitor.
type IngressMonitorMetric struct {
	Namespace string
	Name      string
	Success   bool
}

// AddIngressMonitor adds an extra IngressMonitor to the IngressMonitor Gauge
// for the namespace it's created in.
func (m *Metrics) AddIngressMonitor(obj IngressMonitorMetric) {
	m.ingressMonitorTotalGauge.WithLabelValues(obj.Namespace).Inc()
}

// DeleteIngressMonitor deletes an IngressMonitor to the IngressMonitor Gauge from
// the namespace it's deleted from.
func (m *Metrics) DeleteIngressMonitor(obj IngressMonitorMetric) {
	m.ingressMonitorTotalGauge.WithLabelValues(obj.Namespace).Dec()
}

// SyncIngressMonitor sets up the metrics for a sync action for an
// IngressMonitorMetric.
func (m *Metrics) SyncIngressMonitor(obj IngressMonitorMetric) {
	if obj.Success {
		m.ingressMonitorSuccessGauge.WithLabelValues(obj.Namespace).Inc()
	} else {
		m.ingressMonitorFailedGauge.WithLabelValues(obj.Namespace).Inc()
	}

	m.ingressMonitorSyncGauge.WithLabelValues(obj.Namespace).Inc()
}

// New returns a new metrics handler which registers all it's metrics with the
// specified prometheus Registry to broadcast it's captured values.
func New(reg *prometheus.Registry) *Metrics {
	m := &Metrics{
		ingressMonitorTotalGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: ingressMonitorTotalGauge,
				Help: "Total number of Ingress Monitors in the cluster",
			},
			[]string{"namespace"},
		),

		ingressMonitorSyncGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: ingressMonitorSyncGauge,
				Help: "Total number of sync operations performed on the Ingress Monitors",
			},
			[]string{"namespace"},
		),

		ingressMonitorFailedGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: ingressMonitorFailedGauge,
				Help: "Total number of failed syncs for the Ingress Monitors",
			},
			[]string{"namespace"},
		),

		ingressMonitorSuccessGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: ingressMonitorSuccessGauge,
				Help: "Total number of successful syncs for the Ingress Monitors",
			},
			[]string{"namespace"},
		),
	}

	m.register(reg)
	return m
}

func (m *Metrics) register(reg *prometheus.Registry) {
	reg.MustRegister(
		m.ingressMonitorTotalGauge,
		m.ingressMonitorSyncGauge,
		m.ingressMonitorFailedGauge,
		m.ingressMonitorSuccessGauge,
	)
}
