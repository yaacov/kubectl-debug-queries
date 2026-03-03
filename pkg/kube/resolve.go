package kube

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

// resourceAlias maps common resource type strings (lowercase) to a canonical kind.
var resourceAlias = map[string]string{
	"deploy":       "Deployment",
	"deployment":   "Deployment",
	"deployments":  "Deployment",
	"sts":          "StatefulSet",
	"statefulset":  "StatefulSet",
	"statefulsets": "StatefulSet",
	"ds":           "DaemonSet",
	"daemonset":    "DaemonSet",
	"daemonsets":   "DaemonSet",
	"rs":           "ReplicaSet",
	"replicaset":   "ReplicaSet",
	"replicasets":  "ReplicaSet",
	"job":          "Job",
	"jobs":         "Job",
}

// parseResourceName splits a "type/name" string into (kind, name).
// If there is no slash, it returns ("", name) indicating a plain pod name.
func parseResourceName(name string) (kind, resourceName string) {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", name
	}
	return parts[0], parts[1]
}

// knownSidecars are container names typically injected by service meshes,
// log collectors, and other infrastructure. They are deprioritized when
// auto-selecting a container.
var knownSidecars = map[string]bool{
	"istio-proxy": true, "envoy": true, "envoy-sidecar": true,
	"linkerd-proxy": true, "jaeger-agent": true, "oauth-proxy": true,
	"vault-agent": true, "vault-agent-init": true,
	"filebeat": true, "fluentd": true, "fluentbit": true, "fluent-bit": true,
	"promtail": true, "vector": true, "log-collector": true,
	"kube-rbac-proxy": true,
}

// ResolvePodName resolves the name argument to a concrete pod name.
// If name contains a slash (e.g. "deployment/nginx"), it finds a running pod
// owned by that workload. Otherwise it returns the name unchanged (plain pod).
func ResolvePodName(ctx context.Context, clients *Clients, name, namespace string) (string, error) {
	kind, resourceName := parseResourceName(name)
	if kind == "" {
		return name, nil
	}

	canonical, ok := resourceAlias[strings.ToLower(kind)]
	if !ok {
		return "", fmt.Errorf("unsupported resource type %q for logs; supported: deployment, statefulset, daemonset, replicaset, job", kind)
	}

	klog.V(2).Infof("[resolve] resolving %s/%s in namespace %s", canonical, resourceName, namespace)

	selector, err := workloadSelector(ctx, clients, canonical, resourceName, namespace)
	if err != nil {
		return "", fmt.Errorf("resolving %s/%s: %w", kind, resourceName, err)
	}

	pods, err := clients.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return "", fmt.Errorf("listing pods for %s/%s: %w", kind, resourceName, err)
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for %s/%s in namespace %s", kind, resourceName, namespace)
	}

	// Prefer a Running pod; fall back to the first one.
	for _, p := range pods.Items {
		if p.Status.Phase == "Running" {
			klog.V(2).Infof("[resolve] selected pod %s (Running)", p.Name)
			return p.Name, nil
		}
	}

	klog.V(2).Infof("[resolve] no running pod found; using %s (%s)", pods.Items[0].Name, pods.Items[0].Status.Phase)
	return pods.Items[0].Name, nil
}

// ResolveContainer picks the best container when the user didn't specify one.
// Returns "" if the pod has a single container (Kubernetes handles it).
// For multi-container pods it filters out known sidecars and prefers the first
// application container.
func ResolveContainer(ctx context.Context, clients *Clients, podName, namespace string) (string, error) {
	pod, err := clients.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("fetching pod %s: %w", podName, err)
	}

	containers := pod.Spec.Containers
	if len(containers) <= 1 {
		return "", nil
	}

	// Collect non-sidecar containers.
	var primary []string
	for _, c := range containers {
		if !knownSidecars[c.Name] {
			primary = append(primary, c.Name)
		}
	}

	if len(primary) == 1 {
		klog.V(2).Infof("[resolve] auto-selected container %q (only non-sidecar)", primary[0])
		return primary[0], nil
	}

	// If all were filtered (unlikely) or multiple remain, fall back to the
	// first non-sidecar or the very first container.
	if len(primary) == 0 {
		chosen := containers[0].Name
		klog.V(2).Infof("[resolve] auto-selected container %q (first container, all matched sidecar list)", chosen)
		return chosen, nil
	}

	chosen := primary[0]
	names := make([]string, len(containers))
	for i, c := range containers {
		names[i] = c.Name
	}
	klog.V(2).Infof("[resolve] auto-selected container %q from %v (first non-sidecar)", chosen, names)
	return chosen, nil
}

// workloadSelector returns the label selector string for the pods managed by a workload.
func workloadSelector(ctx context.Context, clients *Clients, kind, name, namespace string) (string, error) {
	switch kind {
	case "Deployment":
		obj, err := clients.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		return metav1.FormatLabelSelector(obj.Spec.Selector), nil

	case "StatefulSet":
		obj, err := clients.Clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		return metav1.FormatLabelSelector(obj.Spec.Selector), nil

	case "DaemonSet":
		obj, err := clients.Clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		return metav1.FormatLabelSelector(obj.Spec.Selector), nil

	case "ReplicaSet":
		obj, err := clients.Clientset.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		return metav1.FormatLabelSelector(obj.Spec.Selector), nil

	case "Job":
		obj, err := clients.Clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		if obj.Spec.Selector == nil {
			return labels.Set(obj.Spec.Template.Labels).String(), nil
		}
		return metav1.FormatLabelSelector(obj.Spec.Selector), nil

	default:
		return "", fmt.Errorf("unsupported workload kind %q", kind)
	}
}
