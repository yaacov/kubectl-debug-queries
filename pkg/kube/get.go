package kube

import (
	"context"
	"fmt"

	ptable "github.com/yaacov/kubectl-debug-queries/pkg/table"
)

// Get retrieves a single resource by type, name, and namespace,
// returning it formatted as a server-side table. The queryStr applies
// TSL-based filtering and field selection to the result.
func Get(ctx context.Context, clients *Clients, resource, name, namespace, format, queryStr string) (string, error) {
	if resource == "" {
		return "", fmt.Errorf("resource is required")
	}
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	gvr, err := clients.resolveGVR(resource)
	if err != nil {
		return "", err
	}

	url := buildResourceURL(clients.Config.Host, gvr, namespace, name)
	tbl, err := clients.doTableRequest(ctx, url)
	if err != nil {
		return "", err
	}

	opts := ptable.Options{}
	return FormatTable(tbl, format, opts, queryStr, false)
}
