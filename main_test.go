package ResponseTimeBalancer

import (
	"fmt"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// MockK8sClient implements K8sClientInterface for testing.
type MockK8sClient struct {
	pods map[string]string
	mu   sync.RWMutex
}

// Ensure MockK8sClient implements K8sClientInterface
var _ K8sClientInterface = (*MockK8sClient)(nil)

// NewMockK8sClient initializes a mock Kubernetes client with test pods.
func NewMockK8sClient() *MockK8sClient {
	return &MockK8sClient{
		pods: map[string]string{
			"abcd1234": "10.0.0.1",
			"efgh5678": "10.0.0.2",
		},
	}
}

// GetPod returns the pod IP by pod-id.
func (kc *MockK8sClient) GetPod(podID string) (string, bool) {
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	pod, exists := kc.pods[podID]
	return pod, exists
}

// GetRandomPod returns a random pod IP.
func (kc *MockK8sClient) GetRandomPod() string {
	kc.mu.RLock()
	defer kc.mu.RUnlock()

	for _, ip := range kc.pods {
		return ip // Returns the first pod for simplicity
	}
	return ""
}

// AddPod simulates adding a new pod.
func (kc *MockK8sClient) AddPod(ip string) {
	kc.mu.Lock()
	defer kc.mu.Unlock()
	hash := crc32.ChecksumIEEE([]byte(ip))
	kc.pods[fmt.Sprintf("%08x", hash)] = ip
}

// RemovePod simulates removing a pod by its IP.
func (kc *MockK8sClient) RemovePod(ip string) {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	for key, val := range kc.pods {
		if val == ip {
			delete(kc.pods, key)
			break
		}
	}
}

// **Test ServeHTTP method for balancing requests**
func TestK8sBalancer_ServeHTTP(t *testing.T) {
	mockClient := NewMockK8sClient()

	// Mock handler that simply responds with "OK"
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	balancer := &K8sBalancer{
		next:   nextHandler,
		k8s:    mockClient,
		header: "pod-id",
	}

	// **1. Test sticky session (when `pod-id` is provided)**
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("pod-id", "abcd1234")
	rec := httptest.NewRecorder()

	balancer.ServeHTTP(rec, req)

	// Verify that the same pod-id is returned in the response
	if rec.Header().Get("pod-id") != "abcd1234" {
		t.Errorf("Expected pod-id 'abcd1234', got '%s'", rec.Header().Get("pod-id"))
	}

	// **2. Test random pod selection when `pod-id` is missing**
	req2 := httptest.NewRequest("GET", "http://example.com", nil)
	rec2 := httptest.NewRecorder()

	balancer.ServeHTTP(rec2, req2)

	// Verify that a pod-id is set in the response
	podID := rec2.Header().Get("pod-id")
	if podID == "" {
		t.Errorf("Expected pod-id to be set, but got an empty string")
	}
}

// **Test pod hash generation**
func TestPodHashGeneration(t *testing.T) {
	podIP := "192.168.1.1"
	expectedHash := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(podIP)))

	if expectedHash == "" {
		t.Errorf("Hash should not be empty")
	}
}

// **Test adding a pod dynamically**
func TestK8sClient_AddPod(t *testing.T) {
	mockClient := NewMockK8sClient()

	mockClient.AddPod("10.0.0.3")

	// Compute the expected hash
	expectedHash := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte("10.0.0.3")))

	if _, exists := mockClient.GetPod(expectedHash); !exists {
		t.Errorf("Expected newly added pod to be retrievable")
	}
}

// **Test removing a pod dynamically**
func TestK8sClient_RemovePod(t *testing.T) {
	mockClient := NewMockK8sClient()

	mockClient.RemovePod("10.0.0.2")

	// Compute the expected hash
	expectedHash := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte("10.0.0.2")))

	if _, exists := mockClient.GetPod(expectedHash); exists {
		t.Errorf("Expected pod to be removed, but it still exists")
	}
}

// **Test fetching a random pod**
func TestK8sClient_GetRandomPod(t *testing.T) {
	mockClient := NewMockK8sClient()

	randomPod := mockClient.GetRandomPod()
	if randomPod == "" {
		t.Errorf("Expected a random pod, but got an empty string")
	}
}
