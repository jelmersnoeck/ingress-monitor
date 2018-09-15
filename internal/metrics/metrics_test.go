package metrics

import (
	"reflect"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	mprom "github.com/prometheus/client_model/go"
)

func TestMetrics_IngressMonitor(t *testing.T) {
	t.Run("adding an ingress monitor", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		m := New(reg)
		m.AddIngressMonitor(IngressMonitorMetric{Namespace: "testing"})

		gatherers := prometheus.Gatherers{
			reg,
			prometheus.DefaultGatherer,
		}

		gathering, err := gatherers.Gather()
		if err != nil {
			t.Fatal(err)
		}

		var testMetric []*mprom.Metric
		for _, gath := range gathering {
			if gath.GetName() == ingressMonitorTotalGauge {
				testMetric = gath.Metric
			}
		}

		exp := []*mprom.Metric{
			{
				Label: []*mprom.LabelPair{labelPair("namespace", "testing")},
				Gauge: &mprom.Gauge{Value: ptrFloat64(1)},
			},
		}

		if !reflect.DeepEqual(testMetric, exp) {
			t.Errorf("Gathered metric\n\n%#v\n\n doesn't equal expected metric\n\n%#v\n\n", testMetric, exp)
		}
	})

	t.Run("deleting an ingress monitor", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		m := New(reg)
		m.DeleteIngressMonitor(IngressMonitorMetric{Namespace: "testing"})

		gatherers := prometheus.Gatherers{
			reg,
			prometheus.DefaultGatherer,
		}

		gathering, err := gatherers.Gather()
		if err != nil {
			t.Fatal(err)
		}

		var testMetric []*mprom.Metric
		for _, gath := range gathering {
			if gath.GetName() == ingressMonitorTotalGauge {
				testMetric = gath.Metric
			}
		}

		exp := []*mprom.Metric{
			{
				Label: []*mprom.LabelPair{labelPair("namespace", "testing")},
				Gauge: &mprom.Gauge{Value: ptrFloat64(-1)},
			},
		}

		if !reflect.DeepEqual(testMetric, exp) {
			t.Errorf("Gathered metric\n\n%#v\n\n doesn't equal expected metric\n\n%#v\n\n", testMetric, exp)
		}
	})

	t.Run("syncing an ingress monitor", func(t *testing.T) {
		tests := []struct {
			name   string
			imm    IngressMonitorMetric
			gm     string
			metric []*mprom.Metric
		}{
			{
				name: "successfully synced",
				imm:  IngressMonitorMetric{Namespace: "testing", Success: true},
				gm:   ingressMonitorSuccessGauge,
				metric: []*mprom.Metric{
					{
						Label: []*mprom.LabelPair{labelPair("namespace", "testing")},
						Gauge: &mprom.Gauge{Value: ptrFloat64(1)},
					},
				},
			},
			{
				name: "unsuccessfully synced",
				imm:  IngressMonitorMetric{Namespace: "testing", Success: false},
				gm:   ingressMonitorFailedGauge,
				metric: []*mprom.Metric{
					{
						Label: []*mprom.LabelPair{labelPair("namespace", "testing")},
						Gauge: &mprom.Gauge{Value: ptrFloat64(1)},
					},
				},
			},
			{
				name: "successfully synced",
				imm:  IngressMonitorMetric{Namespace: "testing", Success: true},
				gm:   ingressMonitorSyncGauge,
				metric: []*mprom.Metric{
					{
						Label: []*mprom.LabelPair{labelPair("namespace", "testing")},
						Gauge: &mprom.Gauge{Value: ptrFloat64(1)},
					},
				},
			},
			{
				name: "unsuccessfully synced",
				imm:  IngressMonitorMetric{Namespace: "testing", Success: false},
				gm:   ingressMonitorSyncGauge,
				metric: []*mprom.Metric{
					{
						Label: []*mprom.LabelPair{labelPair("namespace", "testing")},
						Gauge: &mprom.Gauge{Value: ptrFloat64(1)},
					},
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				reg := prometheus.NewRegistry()
				m := New(reg)
				m.SyncIngressMonitor(test.imm)

				gatherers := prometheus.Gatherers{
					reg,
					prometheus.DefaultGatherer,
				}

				gathering, err := gatherers.Gather()
				if err != nil {
					t.Fatal(err)
				}

				var testMetric []*mprom.Metric
				for _, gath := range gathering {
					if gath.GetName() == test.gm {
						testMetric = test.metric
					}
				}

				if !reflect.DeepEqual(testMetric, test.metric) {
					t.Errorf("Gathered metric\n\n%#v\n\n doesn't equal expected metric\n\n%#v\n\n", testMetric, test.metric)
				}
			})
		}
	})
}

func labelPair(name, value string) *mprom.LabelPair {
	return &mprom.LabelPair{Name: ptrString(name), Value: ptrString(value)}
}

func ptrString(s string) *string {
	return &s
}

func ptrFloat64(f float64) *float64 {
	return &f
}
