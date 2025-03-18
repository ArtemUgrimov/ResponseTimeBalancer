package ResponseTimeBalancer

import (
	"context"
	"fmt"
	"hash/crc32"
	"net/http"
	"os"
)

// K8sBalancer is the middleware struct
// It ensures sticky sessions by routing requests to the same pod using a custom header (pod-id)
type K8sBalancer struct {
	next   http.Handler
	k8s    K8sClientInterface
	header string
}

// New creates a new instance of the middleware
// It initializes the Kubernetes client to fetch pod information dynamically
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	k8sClient, err := NewK8sClient(config.KubernetesNamespace, config.ServiceName)
	if err != nil {
		return nil, err
	}

	return &K8sBalancer{
		next:   next,
		k8s:    k8sClient,
		header: "pod-id",
	}, nil
}

// ServeHTTP handles the request and selects a pod based on the pod-id header
// If the pod-id is missing, it picks a random pod and assigns a new pod-id
// If the pod-id exists but the pod is unavailable, it selects a new one
func (b *K8sBalancer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	podID := req.Header.Get(b.header)
	var targetPod string

	// Case 1: pod-id is missing, pick a random pod
	if podID == "" {
		targetPod = b.k8s.GetRandomPod()
		if targetPod == "" {
			os.Stderr.WriteString("RTB : error: no available pods\n")
			http.Error(rw, "Service Unavailable", http.StatusServiceUnavailable)
			return // Stop further execution
		}
		os.Stderr.WriteString(fmt.Sprintf("RTB : picked %s because input is empty\n", targetPod))
	} else if pod, exists := b.k8s.GetPod(podID); exists {
		// Case 2: pod-id exists and the pod is available
		targetPod = pod
		os.Stderr.WriteString(fmt.Sprintf("RTB : picked %s\n", targetPod))
	} else {
		// Case 3: pod-id exists, but the pod is no longer available
		targetPod = b.k8s.GetRandomPod()
		if targetPod == "" {
			os.Stderr.WriteString(fmt.Sprintf("RTB : error: pod %s does not exist and no available pods\n", podID))
			http.Error(rw, "Service Unavailable", http.StatusServiceUnavailable)
			return // Stop further execution
		}
		os.Stderr.WriteString(fmt.Sprintf("RTB : pod %s does not exist, picked %s\n", podID, targetPod))
	}

	// Generate a new pod-id
	hash := crc32.ChecksumIEEE([]byte(targetPod))
	podID = fmt.Sprintf("%08x", hash)

	// Set headers to enforce sticky session behavior
	req.Host = targetPod
	req.URL.Host = targetPod
	req.URL.Scheme = "http"

	req.Header.Set("X-Forwarded-Host", targetPod)
	req.Header.Set("X-Forwarded-For", req.RemoteAddr)
	req.Header.Set("X-Balancer", "K8sBalancer")

	// Return the updated pod-id to the client
	rw.Header().Set(b.header, podID)

	// Forward the request to the next middleware
	b.next.ServeHTTP(rw, req)
}
