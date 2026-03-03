package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yaacov/kubectl-debug-queries/pkg/connection"
	"github.com/yaacov/kubectl-debug-queries/pkg/kube"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Retrieve container logs",
	Long: `Retrieve logs from a pod or workload.

The --name flag accepts a plain pod name ("my-pod") or a resource reference
("deployment/nginx", "statefulset/web", "daemonset/agent", "replicaset/web-abc",
"job/batch-1"). For non-pod resources, a running pod is automatically selected.

Supports tail lines, time-based filtering, previous container logs,
and reverse-time sorting. All parameters are named flags.

By default, logs are auto-detected and rendered in a compact format.
Use --format raw for unprocessed output or --format json for structured output.

Use --query/-q for TSL filtering on parsed log fields:
  --query "where level = 'ERROR'"
  --query "where message ~= 'timeout'"
  --query "select timestamp, level, message where level = 'ERROR'" --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		namespace, _ := cmd.Flags().GetString("namespace")
		container, _ := cmd.Flags().GetString("container")
		previous, _ := cmd.Flags().GetBool("previous")
		tail, _ := cmd.Flags().GetInt("tail")
		since, _ := cmd.Flags().GetString("since")
		sortBy, _ := cmd.Flags().GetString("sort-by")
		format, _ := cmd.Flags().GetString("format")
		queryStr, _ := cmd.Flags().GetString("query")

		cfg := connection.ResolveRESTConfig(cmd.Context())
		if cfg == nil {
			return fmt.Errorf("no Kubernetes credentials available; provide --kubeconfig or --token")
		}

		clients, err := kube.NewClients(cfg)
		if err != nil {
			return err
		}

		result, err := kube.Logs(cmd.Context(), clients, name, namespace, container, previous, tail, since, sortBy, format, queryStr)
		return outputResult(result, err, format)
	},
}

func init() {
	logsCmd.Flags().String("name", "", "Pod name or resource/name (e.g. deployment/nginx)")
	logsCmd.Flags().String("namespace", "", "Namespace")
	logsCmd.Flags().String("container", "", "Container name")
	logsCmd.Flags().Bool("previous", false, "Return logs from the previous terminated container")
	logsCmd.Flags().Int("tail", 0, "Number of lines from the end to return")
	logsCmd.Flags().String("since", "", "Return logs newer than this duration (e.g. 1h, 30m)")
	logsCmd.Flags().String("sort-by", "", "Sort order: time (default, oldest first) or time_desc (newest first)")
	logsCmd.Flags().String("format", "", "Output format: smart (default), raw, json")
	logsCmd.Flags().StringP("query", "q", "", "TSL query on parsed log fields (e.g. \"where level = 'ERROR'\")")
	_ = logsCmd.MarkFlagRequired("name")
	_ = logsCmd.MarkFlagRequired("namespace")
	rootCmd.AddCommand(logsCmd)
}
