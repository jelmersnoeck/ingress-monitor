package cmd

import (
	"log"
	"time"

	"github.com/jelmersnoeck/ingress-monitor/internal/ingressmonitor"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider/logger"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider/statuscake"
	"github.com/jelmersnoeck/ingress-monitor/internal/signals"
	"github.com/jelmersnoeck/ingress-monitor/pkg/client/generated/clientset/versioned"

	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var operatorFlags struct {
	Namespace    string
	MasterURL    string
	KubeConfig   string
	ResyncPeriod string
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
		log.Fatalf("Error parsing ResyncPeriod: %s", err)
	}

	cfg, err := clientcmd.BuildConfigFromFlags(operatorFlags.MasterURL, operatorFlags.KubeConfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building Kubernetes clientset: %s", err)
	}

	imClient, err := versioned.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building IngressMonitor clientset: %s", err)
	}

	fact := provider.NewFactory(kubeClient)
	statuscake.Register(fact)
	logger.Register(fact)

	op, err := ingressmonitor.NewOperator(
		kubeClient, imClient, operatorFlags.Namespace,
		resync, fact,
	)
	if err != nil {
		log.Fatalf("Error building IngressMonitor Operator: %s", err)
	}

	if err := op.Run(stopCh); err != nil {
		log.Fatalf("Error running the operator: %s", err)
	}
}

func init() {
	rootCmd.AddCommand(operatorCmd)

	operatorCmd.PersistentFlags().StringVarP(&operatorFlags.Namespace, "namespace", "n", v1.NamespaceAll, "The namespace to watch for installed CRDs.")
	operatorCmd.PersistentFlags().StringVar(&operatorFlags.MasterURL, "master-url", "", "The URL of the master API.")
	operatorCmd.PersistentFlags().StringVar(&operatorFlags.KubeConfig, "kubeconfig", "", "Kubeconfig which should be used to talk to the API.")
	operatorCmd.PersistentFlags().StringVar(&operatorFlags.ResyncPeriod, "resync-period", "30s", "Resyncing period to ensure all monitors are up to date.")
}
