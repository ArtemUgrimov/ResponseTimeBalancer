package ResponseTimeBalancer

import (
	"context"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// K8sClient interacts with the Kubernetes API
type K8sClient struct {
	clientset *kubernetes.Clientset
	namespace string
	service   string
	pods      map[string]string
	mu        sync.RWMutex
}

// NewK8sClient initializes the Kubernetes client
func NewK8sClient(namespace, service string, updateInterval time.Duration) (*K8sClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	kc := &K8sClient{
		clientset: clientset,
		namespace: namespace,
		service:   service,
		pods:      make(map[string]string),
	}

	// Start periodic updates
	go kc.updatePodsPeriodically(updateInterval)

	return kc, nil
}

// updatePods fetches the list of pods for the service
func (kc *K8sClient) updatePods() error {
	endpoints, err := kc.clientset.CoreV1().Endpoints(kc.namespace).Get(context.TODO(), kc.service, metav1.GetOptions{})
	if err != nil {
		return err
	}

	kc.mu.Lock()
	defer kc.mu.Unlock()

	kc.pods = make(map[string]string)

	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			podHash := fmt.Sprintf("%x", addr.IP)[:8] // Generate short hash
			kc.pods[podHash] = addr.IP
		}
	}

	return nil
}

// updatePodsPeriodically runs in a loop to update the pod list
func (kc *K8sClient) updatePodsPeriodically(interval time.Duration) {
	for {
		if err := kc.updatePods(); err != nil {
			fmt.Println("Failed to update pod list:", err)
		}
		time.Sleep(interval)
	}
}

// GetPod returns the IP of a pod by its hash
func (kc *K8sClient) GetPod(podID string) (string, bool) {
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	pod, exists := kc.pods[podID]
	return pod, exists
}

// GetRandomPod returns a random pod IP
func (kc *K8sClient) GetRandomPod() string {
	kc.mu.RLock()
	defer kc.mu.RUnlock()

	for _, ip := range kc.pods {
		return ip // Simple round-robin
	}

	return ""
}
