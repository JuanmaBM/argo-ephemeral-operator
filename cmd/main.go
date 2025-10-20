package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	ephemeralv1alpha1 "github.com/jbarea/argo-ephemeral-operator/api/v1alpha1"
	"github.com/jbarea/argo-ephemeral-operator/internal/argocd"
	"github.com/jbarea/argo-ephemeral-operator/internal/config"
	"github.com/jbarea/argo-ephemeral-operator/internal/controller"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ephemeralv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		setupLog.Error(err, "unable to load configuration")
		os.Exit(1)
	}

	setupLog.Info("starting argo-ephemeral-operator",
		"argoServer", cfg.ArgoServer,
		"argoNamespace", cfg.ArgoNamespace,
		"reconcileInterval", cfg.ReconcileInterval,
	)

	// Override config with command line flags if provided
	if metricsAddr != "" {
		cfg.MetricsAddr = metricsAddr
	}
	if probeAddr != "" {
		cfg.ProbeAddr = probeAddr
	}
	cfg.EnableLeaderElection = enableLeaderElection

	// Create manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:           scheme,
		LeaderElection:   cfg.EnableLeaderElection,
		LeaderElectionID: cfg.LeaderElectionID,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create ArgoCD client
	argoClient := argocd.NewClient(mgr.GetClient(), cfg.ArgoNamespace)

	// Create Application builder
	appBuilder := argocd.NewApplicationBuilder(scheme)

	// Setup reconciler
	if err = (&controller.EphemeralApplicationReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		ArgoClient:    argoClient,
		AppBuilder:    appBuilder,
		Config:        cfg,
		NameGenerator: &controller.DefaultNameGenerator{},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EphemeralApplication")
		os.Exit(1)
	}

	// Add health and ready checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
