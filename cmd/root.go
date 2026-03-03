// Package cmd implements the cobra CLI commands for kubectl-debug-queries.
package cmd

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yaacov/kubectl-debug-queries/pkg/connection"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

var configFlags *genericclioptions.ConfigFlags

var rootCmd = &cobra.Command{
	Use:   "kubectl-debug-queries",
	Short: "Query Kubernetes resources, logs, and events",
	Long: `Query Kubernetes resources, logs, and events on Kubernetes/OpenShift clusters.

It provides a CLI and an MCP server, both backed by shared logic.

Authentication:
  Standard kubectl flags (--kubeconfig, --context, --token, --server, etc.)

All commands use named flags only — no positional arguments.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config, err := configFlags.ToRESTConfig()
		if err != nil {
			klog.V(2).Infof("Could not load kubeconfig: %v", err)
			return
		}

		logAuthMethod(config)

		cfg := rest.CopyConfig(config)
		cfg.TLSClientConfig.Insecure = true
		cfg.TLSClientConfig.CAData = nil
		cfg.TLSClientConfig.CAFile = ""

		connection.SetDefaultRESTConfig(cfg)
	},
}

func logAuthMethod(config *rest.Config) {
	klog.V(2).Infof("[auth] API server: %s", config.Host)
	switch {
	case config.BearerToken != "":
		klog.V(2).Infof("[auth] Method: bearer token (length %d)", len(config.BearerToken))
	case config.BearerTokenFile != "":
		klog.V(2).Infof("[auth] Method: bearer token file (%s)", config.BearerTokenFile)
	case config.ExecProvider != nil:
		klog.V(2).Infof("[auth] Method: exec provider (%s)", config.ExecProvider.Command)
	case config.CertData != nil || config.CertFile != "":
		klog.V(2).Infof("[auth] Method: client certificate")
	case config.Username != "":
		klog.V(2).Infof("[auth] Method: basic auth (user: %s)", config.Username)
	case config.AuthProvider != nil:
		klog.V(2).Infof("[auth] Method: auth provider (%s)", config.AuthProvider.Name)
	default:
		klog.V(2).Info("[auth] WARNING: no authentication credentials found in REST config")
	}
}

// BuildInsecureTransport creates an http.RoundTripper from the REST config
// that carries kubeconfig credentials but uses InsecureSkipVerify.
func BuildInsecureTransport(config *rest.Config) (http.RoundTripper, error) {
	promConfig := rest.CopyConfig(config)
	promConfig.TLSClientConfig.Insecure = true
	promConfig.TLSClientConfig.CAData = nil
	promConfig.TLSClientConfig.CAFile = ""
	return rest.TransportFor(promConfig)
}

func init() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	configFlags = genericclioptions.NewConfigFlags(true)
	configFlags.AddFlags(rootCmd.PersistentFlags())
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
