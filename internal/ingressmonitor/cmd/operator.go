package cmd

import (
	"os"
	"time"

	"github.com/jelmersnoeck/ingress-monitor/internal/httpsvc"
	"github.com/jelmersnoeck/ingress-monitor/internal/ingressmonitor"
	"github.com/jelmersnoeck/ingress-monitor/internal/metrics"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider/logger"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider/statuscake"
	"github.com/jelmersnoeck/ingress-monitor/internal/signals"
	"github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var operatorFlags struct {
	Namespace    string
	MasterURL    string
	KubeConfig   string
	ResyncPeriod string

	MetricsAddr string
	MetricsPort int
}

// operatorCmd represents the operator command
var operatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Run the IngressMonitor Operator",
	Run:   runOperator,
}

func runOperator(cmd *cobra.Command, args []string) {
	stopCh := signals.SetupSignalHandler()

	resync, err := time.ParseDuration(operatorFlags.ResyncPeriod)
	if err != nil {
		logrus.WithError(err).Fatal("Error parsing ResyncPeriod")
	}

	cfg, err := clientcmd.BuildConfigFromFlags(operatorFlags.MasterURL, operatorFlags.KubeConfig)
	if err != nil {
		logrus.WithError(err).Fatal("Error building kubeconfig")
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Error building Kubernetes clientset")
	}

	imClient, err := versioned.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Error building IngressMonitor clientset")
	}

	// register the available providers
	fact := provider.NewFactory(kubeClient)
	statuscake.Register(fact)
	logger.Register(fact)

	// create new prometheus registry
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewProcessCollector(os.Getpid(), ""))
	registry.MustRegister(prometheus.NewGoCollector())

	// new metrics collector
	mtrc := metrics.New(registry)
	metricssvc := httpsvc.Metrics{
		Server: httpsvc.Server{
			Addr: operatorFlags.MetricsAddr,
			Port: operatorFlags.MetricsPort,
		},
		Registry: registry,
	}
	go metricssvc.Start(stopCh)

	op, err := ingressmonitor.NewOperator(
		kubeClient, imClient, operatorFlags.Namespace,
		resync, fact, mtrc,
	)
	if err != nil {
		logrus.WithError(err).Fatalf("Error building IngressMonitor Operator")
	}

	if err := op.Run(stopCh); err != nil {
		logrus.WithError(err).Fatalf("Error running the operator")
	}
}

func init() {
	rootCmd.AddCommand(operatorCmd)

	operatorCmd.PersistentFlags().StringVarP(&operatorFlags.Namespace, "namespace", "n", v1.NamespaceAll, "The namespace to watch for installed CRDs.")
	operatorCmd.PersistentFlags().StringVar(&operatorFlags.MasterURL, "master-url", "", "The URL of the master API.")
	operatorCmd.PersistentFlags().StringVar(&operatorFlags.KubeConfig, "kubeconfig", "", "Kubeconfig which should be used to talk to the API.")
	operatorCmd.PersistentFlags().StringVar(&operatorFlags.ResyncPeriod, "resync-period", "30s", "Resyncing period to ensure all monitors are up to date.")

	operatorCmd.PersistentFlags().StringVar(&operatorFlags.MetricsAddr, "metrics-addr", "0.0.0.0", "address the metrics server will bind to")
	operatorCmd.PersistentFlags().IntVar(&operatorFlags.MetricsPort, "metrics-port", 9090, "port on which the metrics server is available")
}
