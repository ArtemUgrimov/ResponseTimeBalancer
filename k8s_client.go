package ResponseTimeBalancer

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
)

// K8sClientInterface defines the contract for Kubernetes client interactions.
type K8sClientInterface interface {
	GetPod(podID string) (string, bool)
	GetRandomPod() string
}

// Ensure K8sClient implements K8sClientInterface
var _ K8sClientInterface = (*K8sClient)(nil)

// K8sClient interacts with Kubernetes API using HTTP requests.
type K8sClient struct {
	apiURL     string
	token      string
	namespace  string
	service    string
	pods       map[string]string
	mu         sync.RWMutex
	httpClient *http.Client
}

// NewK8sClient initializes the Kubernetes client, loads initial pod list, and starts watching changes.
func NewK8sClient(namespace, service string) (*K8sClient, error) {
	// Read the token from the service account file
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, fmt.Errorf("RTB : cannot read service account token: %s", err.Error())
	}

	// Get Kubernetes API URL from environment variables
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return nil, fmt.Errorf("RTB : missing Kubernetes API host/port environment variables")
	}

	apiURL := fmt.Sprintf("https://%s:%s", host, port)

	// Initialize K8sClient
	kc := &K8sClient{
		apiURL:    apiURL,
		token:     strings.TrimSpace(string(token)),
		namespace: namespace,
		service:   service,
		pods:      make(map[string]string),
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}

	// Load full list of pods at startup
	if err := kc.updatePods(); err != nil {
		return nil, fmt.Errorf("RTB : failed to fetch initial pod list: %s", err.Error())
	}

	// Start watching for pod changes
	go kc.watchPods()

	return kc, nil
}

// updatePods fetches the full list of pods using Kubernetes API.
func (kc *K8sClient) updatePods() error {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods?labelSelector=app=%s", kc.apiURL, kc.namespace, kc.service)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+kc.token)
	req.Header.Set("Accept", "application/json")

	resp, err := kc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("RTB : failed to fetch pods, status: %d\nbody: %s\nurl: %s\n", resp.StatusCode, string(body), url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse JSON response
	var result struct {
		Items []struct {
			Status struct {
				PodIP string `json:"podIP"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	kc.mu.Lock()
	defer kc.mu.Unlock()

	kc.pods = make(map[string]string)

	ips := ""
	for _, pod := range result.Items {
		if pod.Status.PodIP != "" {
			hash := crc32.ChecksumIEEE([]byte(pod.Status.PodIP))
			podHash := fmt.Sprintf("%08x", hash)
			kc.pods[podHash] = pod.Status.PodIP
			ips += pod.Status.PodIP + "; "
		}
	}
	os.Stderr.WriteString(fmt.Sprintf("RTB : Initial pod list loaded: len=%d, ips: %s\n", len(kc.pods), ips))

	return nil
}

// watchPods listens for pod changes in Kubernetes, adding only Ready pods.
func (kc *K8sClient) watchPods() {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods?watch=true&labelSelector=app=%s", kc.apiURL, kc.namespace, kc.service)

	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "RTB : Failed to create watch request: %s\n", err.Error())
			continue
		}
		req.Header.Set("Authorization", "Bearer "+kc.token)
		req.Header.Set("Accept", "application/json")

		resp, err := kc.httpClient.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "RTB : Failed to execute watch request: %s\n", err.Error())
			continue
		}
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)

		// Stream updates in real time
		for {
			var event struct {
				Type   string `json:"type"`
				Object struct {
					Status struct {
						PodIP      string `json:"podIP"`
						Conditions []struct {
							Type   string `json:"type"`
							Status string `json:"status"`
						} `json:"conditions"`
					} `json:"status"`
				} `json:"object"`
			}

			if err := decoder.Decode(&event); err == io.EOF {
				break
			} else if err != nil {
				fmt.Fprintf(os.Stderr, "RTB : Error decoding pod event: %s\n", err.Error())
				break
			}

			// Determine if the pod is Ready
			isReady := false
			for _, condition := range event.Object.Status.Conditions {
				if condition.Type == "Ready" && condition.Status == "True" {
					isReady = true
					break
				}
			}

			// Lock for all operations
			kc.mu.Lock()

			if event.Type == "ADDED" || event.Type == "MODIFIED" {
				if isReady {
					hash := crc32.ChecksumIEEE([]byte(event.Object.Status.PodIP))
					podHash := fmt.Sprintf("%08x", hash)
					kc.pods[podHash] = event.Object.Status.PodIP
					fmt.Fprintf(os.Stderr, "RTB : Pod added/updated (Ready): %s\n", event.Object.Status.PodIP)
				} else {
					// Remove pod if it's not ready anymore
					for key, ip := range kc.pods {
						if ip == event.Object.Status.PodIP {
							delete(kc.pods, key)
							fmt.Fprintf(os.Stderr, "RTB : Pod removed (Not Ready): %s\n", event.Object.Status.PodIP)
							break
						}
					}
				}
			} else if event.Type == "DELETED" {
				// Remove pod when it is completely deleted
				for key, ip := range kc.pods {
					if ip == event.Object.Status.PodIP {
						delete(kc.pods, key)
						fmt.Fprintf(os.Stderr, "RTB : Pod removed (Deleted): %s\n", event.Object.Status.PodIP)
						break
					}
				}
			}

			kc.mu.Unlock()
		}
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
