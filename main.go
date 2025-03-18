package ResponseTimeBalancer

import (
	"context"
	"fmt"
	"hash/crc32"
	"net/http"
	"os"
)

// K8sBalancer is the middleware struct
type K8sBalancer struct {
	next   http.Handler
	k8s    K8sClientInterface
	header string
}

// New creates a new instance of the middleware
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

// ServeHTTP handles the request and selects a pod
func (b *K8sBalancer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	podID := req.Header.Get(b.header)

	var targetPod string
	if len(podID) == 0 {
		targetPod = b.k8s.GetRandomPod()
		os.Stderr.WriteString(fmt.Sprintf("RTB : picked %s because input is empty\n", targetPod))
	} else if pod, exists := b.k8s.GetPod(podID); exists {
		targetPod = pod
		os.Stderr.WriteString(fmt.Sprintf("RTB : picked %s\n", targetPod))
	} else {
		targetPod = b.k8s.GetRandomPod()
		if len(targetPod) > 0 {
			os.Stderr.WriteString(fmt.Sprintf("RTB : pod %s does not exist, picked %s\n", podID, targetPod))
		} else {
			os.Stderr.WriteString("RTB : pod cannot be picked\n")
		}
	}

	if len(targetPod) > 0 {
		hash := crc32.ChecksumIEEE([]byte(targetPod))
		podID = fmt.Sprintf("%08x", hash)

		req.Host = targetPod
		req.URL.Host = targetPod
		req.URL.Scheme = "http"
		req.Header.Set("X-Forwarded-Host", targetPod)
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		req.Header.Set("X-Balancer", "K8sBalancer")
		rw.Header().Set(b.header, podID)
	}

	b.next.ServeHTTP(rw, req)
}
