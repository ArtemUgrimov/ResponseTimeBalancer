package ResponseTimeBalancer

type Config struct {
	KubernetesNamespace string `json:"namespace,omitempty"`
	ServiceName         string `json:"serviceName,omitempty"`
}

func CreateConfig() *Config {
	return &Config{
		KubernetesNamespace: "rc",
		ServiceName:         "bng-server",
	}
}
