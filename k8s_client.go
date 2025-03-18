package ResponseTimeBalancer

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// K8sClient interacts with Kubernetes API using raw HTTP requests.
type K8sClient struct {
	apiURL     string
	token      string
	namespace  string
	service    string
	pods       map[string]string
	mu         sync.RWMutex
	httpClient *http.Client
}

// NewK8sClient initializes the Kubernetes client using HTTP requests.
func NewK8sClient(namespace, service string, updateInterval time.Duration) (*K8sClient, error) {
	// Read token from service account file
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("RTB : cannot read service account token: %s\n", err.Error()))
		return nil, err
	}

	// Get Kubernetes API URL from environment variables
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return nil, fmt.Errorf("RTB : missing Kubernetes API host/port environment variables")
	}

	apiURL := fmt.Sprintf("https://%s:%s", host, port)

	// Create HTTP client with TLS config
	kc := &K8sClient{
		apiURL:    apiURL,
		token:     strings.TrimSpace(string(token)),
		namespace: namespace,
		service:   service,
		pods:      make(map[string]string),
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Disable verification (better to use CA from mounted file)
				},
			},
			Timeout: 5 * time.Second,
		},
	}

	// Start periodic updates
	go kc.updatePodsPeriodically(updateInterval)

	return kc, nil
}

// updatePods fetches the list of pods for the service using Kubernetes API.
func (kc *K8sClient) updatePods() error {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/endpoints/%s", kc.apiURL, kc.namespace, kc.service)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+kc.token)
	req.Header.Set("Accept", "application/json")

	resp, err := kc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("RTB : failed to fetch endpoints, status: %d\nbody: %s\nurl: %s\n", resp.StatusCode, (string)(body), url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse JSON response
	var result struct {
		Subsets []struct {
			Addresses []struct {
				IP string `json:"ip"`
			} `json:"addresses"`
		} `json:"subsets"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	kc.mu.Lock()
	defer kc.mu.Unlock()

	kc.pods = make(map[string]string)

	for _, subset := range result.Subsets {
		for _, addr := range subset.Addresses {
			podHash := fmt.Sprintf("%x", addr.IP)[:8] // Generate short hash
			kc.pods[podHash] = addr.IP
		}
	}
	os.Stderr.WriteString(fmt.Sprintf("RTB : updated pod list: len=%d\n", len(kc.pods)))

	return nil
}

// updatePodsPeriodically runs in a loop to update the pod list.
func (kc *K8sClient) updatePodsPeriodically(interval time.Duration) {
	for {
		if err := kc.updatePods(); err != nil {
			os.Stderr.WriteString(fmt.Sprintf("RTB : failed to update pod list: %s\n", err.Error()))
		}
		time.Sleep(interval)
	}
}

// GetPod returns the IP of a pod by its hash.
func (kc *K8sClient) GetPod(podID string) (string, bool) {
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	pod, exists := kc.pods[podID]
	return pod, exists
}

// GetRandomPod returns a random pod IP.
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
