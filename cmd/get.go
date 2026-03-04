package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaacov/kubectl-debug-queries/pkg/connection"
	"github.com/yaacov/kubectl-debug-queries/pkg/kube"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a single Kubernetes resource",
	Long: `Get a single Kubernetes resource by type, name, and namespace.

Columns are auto-detected from the API server (same as kubectl get).
All parameters are named flags.

Use --query/-q for field selection with JSON/YAML output:
  --query "select Name, Status" --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resource, _ := cmd.Flags().GetString("resource")
		name, _ := cmd.Flags().GetString("name")
		namespace, _ := cmd.Flags().GetString("namespace")
		format, _ := cmd.Flags().GetString("output")
		queryStr, _ := cmd.Flags().GetString("query")

		cfg := connection.ResolveRESTConfig(cmd.Context())
		if cfg == nil {
			return fmt.Errorf("no Kubernetes credentials available; provide --kubeconfig or --token")
		}

		clients, err := kube.NewClients(cfg)
		if err != nil {
			return err
		}

		result, err := kube.Get(cmd.Context(), clients, resource, name, namespace, format, queryStr)
		return outputResult(result, err, format)
	},
}

func init() {
	getCmd.Flags().String("resource", "", "Resource type (e.g. pod, deployment, service)")
	getCmd.Flags().String("name", "", "Resource name")
	getCmd.Flags().String("namespace", "", "Namespace")
	getCmd.Flags().StringP("output", "o", "table", "Output format: table, markdown, json, yaml")
	getCmd.Flags().StringP("query", "q", "", "TSL query for field selection (e.g. \"select Name, Status\")")
	_ = getCmd.MarkFlagRequired("resource")
	_ = getCmd.MarkFlagRequired("name")
	_ = getCmd.MarkFlagRequired("namespace")
	rootCmd.AddCommand(getCmd)
}
