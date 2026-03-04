package kube

import (
	"context"
	"fmt"

	ptable "github.com/yaacov/kubectl-debug-queries/pkg/table"
)

// List retrieves resources of a given type, optionally filtered by
// namespace and label selector. The queryStr applies TSL-based filtering
// (WHERE, ORDER BY, LIMIT, SELECT) to the results.
func List(ctx context.Context, clients *Clients, resource, namespace, selector, sortBy string, limit int, allNamespaces bool, format, queryStr string) (string, error) {
	if resource == "" {
		return "", fmt.Errorf("resource is required")
	}
	if !allNamespaces && namespace == "" {
		return "", fmt.Errorf("namespace is required (or use all_namespaces)")
	}

	gvr, err := clients.resolveGVR(resource)
	if err != nil {
		return "", err
	}

	ns := namespace
	if allNamespaces {
		ns = ""
	}

	url := buildResourceURL(clients.Config.Host, gvr, ns, "")
	if selector != "" {
		url += "?labelSelector=" + selector
	}

	tbl, err := clients.doTableRequest(ctx, url)
	if err != nil {
		return "", err
	}

	opts := ptable.Options{
		SortBy: sortBy,
		Limit:  limit,
	}
	return FormatTable(tbl, format, opts, queryStr, allNamespaces)
}
