package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaacov/kubectl-debug-queries/pkg/connection"
	"github.com/yaacov/kubectl-debug-queries/pkg/kube"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Kubernetes resources",
	Long: `List Kubernetes resources of a given type.

Columns are auto-detected from the API server. Supports label selectors,
sorting by any column, and row limiting. All parameters are named flags.

Use --query/-q for TSL (Tree Search Language) filtering:
  --query "where Status = 'Running'"
  --query "where Name ~= 'nginx-.*' order by Age desc limit 10"
  --query "select Name, Status where Restarts > 5"

For JSON output, SELECT controls which fields appear in the output.
For table output, the original server-side columns are always shown.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resource, _ := cmd.Flags().GetString("resource")
		namespace, _ := cmd.Flags().GetString("namespace")
		selector, _ := cmd.Flags().GetString("selector")
		sortBy, _ := cmd.Flags().GetString("sort-by")
		limit, _ := cmd.Flags().GetInt("limit")
		allNamespaces, _ := cmd.Flags().GetBool("all-namespaces")
		format, _ := cmd.Flags().GetString("output")
		queryStr, _ := cmd.Flags().GetString("query")

		if !allNamespaces && namespace == "" {
			return fmt.Errorf("--namespace is required (or use --all-namespaces)")
		}

		cfg := connection.ResolveRESTConfig(cmd.Context())
		if cfg == nil {
			return fmt.Errorf("no Kubernetes credentials available; provide --kubeconfig or --token")
		}

		clients, err := kube.NewClients(cfg)
		if err != nil {
			return err
		}

		result, err := kube.List(cmd.Context(), clients, resource, namespace, selector, sortBy, limit, allNamespaces, format, queryStr)
		return outputResult(result, err, format)
	},
}

func init() {
	listCmd.Flags().String("resource", "", "Resource type (e.g. pods, deployments, services)")
	listCmd.Flags().String("namespace", "", "Namespace")
	listCmd.Flags().StringP("selector", "l", "", "Label selector (e.g. app=nginx)")
	listCmd.Flags().String("sort-by", "", "Column name to sort by")
	listCmd.Flags().Int("limit", 0, "Maximum number of rows to return")
	listCmd.Flags().BoolP("all-namespaces", "A", false, "List across all namespaces")
	listCmd.Flags().StringP("output", "o", "table", "Output format: table, markdown, json, yaml")
	listCmd.Flags().StringP("query", "q", "", "TSL query (e.g. \"where Status = 'Running'\", \"select Name, Status where Restarts > 5\")")
	_ = listCmd.MarkFlagRequired("resource")
	rootCmd.AddCommand(listCmd)
}
