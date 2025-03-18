package ResponseTimeBalancer

import (
	"context"
	"fmt"
	"math/rand"
	"os"
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
		os.Stderr.WriteString(fmt.Sprintf("RTB : cannot call InClusterConfig for k8s client: %s\n", err.Error()))
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("RTB : cannot instantiate k8s client: %s\n", err.Error()))
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
	os.Stderr.WriteString(fmt.Sprintf("RTB : updated pod list: len=%d\n", len(kc.pods)))

	return nil
}

// updatePodsPeriodically runs in a loop to update the pod list
func (kc *K8sClient) updatePodsPeriodically(interval time.Duration) {
	for {
		if err := kc.updatePods(); err != nil {
			os.Stderr.WriteString(fmt.Sprintf("RTB : failed to update pod list: %s\n", err.Error()))
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

	if len(kc.pods) == 0 {
		return ""
	}

	var podList []string
	for _, ip := range kc.pods {
		podList = append(podList, ip)
	}

	return podList[rand.Intn(len(podList))]
}
