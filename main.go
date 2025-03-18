package ResponseTimeBalancer

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
)

// K8sBalancer is the middleware struct
type K8sBalancer struct {
	next   http.Handler
	k8s    *K8sClient
	header string
}

// New creates a new instance of the middleware
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	k8sClient, err := NewK8sClient(config.KubernetesNamespace, config.ServiceName, config.UpdateInterval)
	if err != nil {
		return nil, err
	}

	return &K8sBalancer{
		next:   next,
		k8s:    k8sClient,
		header: "pod-id",
	}, nil
}

// ServeHTTP handles the request and selects a pod
func (b *K8sBalancer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	podID := req.Header.Get(b.header)

	var targetPod string
	if pod, exists := b.k8s.GetPod(podID); exists {
		targetPod = pod
	} else {
		targetPod = b.k8s.GetRandomPod()
		podID = fmt.Sprintf("%x", rand.Intn(1000000))[:8]
	}

	req.URL.Host = targetPod
	req.URL.Scheme = "http"
	req.Header.Set("X-Balancer", "K8sBalancer")
	rw.Header().Set(b.header, podID)

	b.next.ServeHTTP(rw, req)
}
