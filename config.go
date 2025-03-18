package ResponseTimeBalancer

import "time"

type Config struct {
	KubernetesNamespace string        `json:"namespace,omitempty"`
	ServiceName         string        `json:"serviceName,omitempty"`
	UpdateInterval      time.Duration `json:"updateInterval,omitempty"`
}

func CreateConfig() *Config {
	return &Config{
		KubernetesNamespace: "rc",
		ServiceName:         "bng-rc",
		UpdateInterval:      time.Second * 30,
	}
}
