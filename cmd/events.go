package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaacov/kubectl-debug-queries/pkg/connection"
	"github.com/yaacov/kubectl-debug-queries/pkg/kube"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "List Kubernetes events",
	Long: `List Kubernetes events, optionally filtered by involved resource.

Columns are auto-detected from the API server. Supports filtering by
resource kind and name, sorting, and row limiting. All parameters are named flags.

Use --query/-q for TSL (Tree Search Language) filtering:
  --query "where Type = 'Warning'"
  --query "where Reason = 'BackOff' order by Last_Seen desc"
  --query "select Reason, Message where Type = 'Warning'" --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")
		resource, _ := cmd.Flags().GetString("resource")
		name, _ := cmd.Flags().GetString("name")
		sortBy, _ := cmd.Flags().GetString("sort-by")
		limit, _ := cmd.Flags().GetInt("limit")
		allNamespaces, _ := cmd.Flags().GetBool("all-namespaces")
		format, _ := cmd.Flags().GetString("format")
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

		result, err := kube.Events(cmd.Context(), clients, namespace, resource, name, sortBy, limit, allNamespaces, format, queryStr)
		return outputResult(result, err, format)
	},
}

func init() {
	eventsCmd.Flags().String("namespace", "", "Namespace")
	eventsCmd.Flags().String("resource", "", "Filter by involved object kind (e.g. Pod, Deployment)")
	eventsCmd.Flags().String("name", "", "Filter by involved object name")
	eventsCmd.Flags().String("sort-by", "", "Column name to sort by")
	eventsCmd.Flags().Int("limit", 0, "Maximum number of rows to return")
	eventsCmd.Flags().Bool("all-namespaces", false, "List events across all namespaces")
	eventsCmd.Flags().String("format", "table", "Output format: table, markdown, json, yaml")
	eventsCmd.Flags().StringP("query", "q", "", "TSL query (e.g. \"where Type = 'Warning'\", \"select Reason, Message\")")
	rootCmd.AddCommand(eventsCmd)
}
