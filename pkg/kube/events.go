package kube

import (
	"context"
	"fmt"
	"net/url"

	"k8s.io/apimachinery/pkg/runtime/schema"

	ptable "github.com/yaacov/kubectl-debug-queries/pkg/table"
)

// Events retrieves events, optionally filtered by namespace, involved
// resource type, and resource name. The queryStr applies TSL-based
// filtering (WHERE, ORDER BY, LIMIT, SELECT) to the results.
func Events(ctx context.Context, clients *Clients, namespace, resource, name, sortBy string, limit int, allNamespaces bool, format, queryStr string) (string, error) {
	if !allNamespaces && namespace == "" {
		return "", fmt.Errorf("namespace is required (or use all_namespaces)")
	}

	ns := namespace
	if allNamespaces {
		ns = ""
	}

	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}
	eventsURL := buildResourceURL(clients.Config.Host, gvr, ns, "")

	var fieldSelectors []string
	if resource != "" {
		fieldSelectors = append(fieldSelectors, "involvedObject.kind="+resource)
	}
	if name != "" {
		fieldSelectors = append(fieldSelectors, "involvedObject.name="+name)
	}

	if len(fieldSelectors) > 0 {
		params := url.Values{}
		params.Set("fieldSelector", joinSelectors(fieldSelectors))
		eventsURL += "?" + params.Encode()
	}

	tbl, err := clients.doTableRequest(ctx, eventsURL)
	if err != nil {
		return "", err
	}

	opts := ptable.Options{
		SortBy: sortBy,
		Limit:  limit,
	}
	return FormatTable(tbl, format, opts, queryStr, allNamespaces)
}

func joinSelectors(selectors []string) string {
	result := selectors[0]
	for i := 1; i < len(selectors); i++ {
		result += "," + selectors[i]
	}
	return result
}
