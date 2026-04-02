// Package kube provides Kubernetes API operations for kubectl-debug-queries.
package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const tableAcceptHeader = "application/json;as=Table;v=v1;g=meta.k8s.io"

// Clients holds the Kubernetes clients built from a rest.Config.
type Clients struct {
	Config    *rest.Config
	Clientset *kubernetes.Clientset
}

// NewClients builds typed and discovery clients from a rest.Config.
func NewClients(cfg *rest.Config) (*Clients, error) {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating clientset: %w", err)
	}
	return &Clients{Config: cfg, Clientset: cs}, nil
}

// ServerTable represents the server-side table response from the K8s API.
type ServerTable struct {
	Columns []metav1.TableColumnDefinition `json:"columnDefinitions"`
	Rows    []TableRow                     `json:"rows"`
}

// TableRow is a row in a server-side table response.
type TableRow struct {
	Cells  []interface{}          `json:"cells"`
	Object map[string]interface{} `json:"object,omitempty"`
}

// resolveGVR resolves a resource string (like "pods", "deployments", "virtualmachines")
// to a GroupVersionResource by querying the API server's discovery endpoint.
func (c *Clients) resolveGVR(resource string) (schema.GroupVersionResource, error) {
	disc := c.Clientset.Discovery()
	apiResources, err := disc.ServerPreferredResources()
	if err != nil {
		if !discovery.IsGroupDiscoveryFailedError(err) {
			return schema.GroupVersionResource{}, fmt.Errorf("API discovery: %w", err)
		}
		klog.V(2).Infof("[discovery] partial failure (some groups unavailable): %v", err)
	}

	lower := strings.ToLower(resource)

	// Support group-qualified names like "plans.forklift.konveyor.io".
	var wantGroup string
	if dot := strings.IndexByte(lower, '.'); dot > 0 {
		wantGroup = lower[dot+1:]
		lower = lower[:dot]
	}

	for _, list := range apiResources {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		if wantGroup != "" && !strings.EqualFold(gv.Group, wantGroup) {
			continue
		}
		for _, r := range list.APIResources {
			if strings.ToLower(r.Name) == lower || strings.ToLower(r.SingularName) == lower {
				return schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: r.Name,
				}, nil
			}
			for _, shortName := range r.ShortNames {
				if strings.ToLower(shortName) == lower {
					return schema.GroupVersionResource{
						Group:    gv.Group,
						Version:  gv.Version,
						Resource: r.Name,
					}, nil
				}
			}
		}
	}
	return schema.GroupVersionResource{}, fmt.Errorf("resource %q not found on the server", resource)
}

// buildResourceURL constructs the REST URL for a resource request.
func buildResourceURL(host string, gvr schema.GroupVersionResource, namespace, name string) string {
	var base string
	if gvr.Group == "" {
		base = fmt.Sprintf("%s/api/%s", strings.TrimRight(host, "/"), gvr.Version)
	} else {
		base = fmt.Sprintf("%s/apis/%s/%s", strings.TrimRight(host, "/"), gvr.Group, gvr.Version)
	}

	if namespace != "" {
		base += "/namespaces/" + namespace
	}
	base += "/" + gvr.Resource
	if name != "" {
		base += "/" + name
	}
	return base
}

// doTableRequest performs an HTTP GET with the server-side table Accept header
// and decodes the response into a ServerTable. Full resource objects are always
// requested (includeObject=Object) so that queries can filter on any field.
func (c *Clients) doTableRequest(ctx context.Context, rawURL string) (*ServerTable, error) {
	rt, err := rest.TransportFor(c.Config)
	if err != nil {
		return nil, fmt.Errorf("building transport: %w", err)
	}

	if strings.Contains(rawURL, "?") {
		rawURL += "&includeObject=Object"
	} else {
		rawURL += "?includeObject=Object"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", tableAcceptHeader)

	resp, err := (&http.Client{Transport: rt}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 500 {
			body = body[:500]
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tbl ServerTable
	if err := json.NewDecoder(resp.Body).Decode(&tbl); err != nil {
		return nil, fmt.Errorf("decoding server table: %w", err)
	}
	return &tbl, nil
}
