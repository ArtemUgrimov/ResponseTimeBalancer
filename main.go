package ResponseTimeBalancer

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
)

// Config the plugin configuration.
type Config struct {
	ResponseTimeHeaderName string `json:"responseTimeHeaderName"`
	ResponseTimeLimitMs    string `json:"responseTimeLimitMs"`
	CookieSetHeaderValue   string `json:"cookieSetHeaderValue"`
	PartitionedHeaderValue string `json:"partitionedHeaderValue"`

	EnableCookieInvalidation bool `json:"enableCookieInvalidation"`
	LogStartup               bool `json:"logStartup"`
	LogSetCookie             bool `json:"logSetCookie"`
	LogLimitNotReached       bool `json:"logLimitNotReached"`
	LogHeaderNotFound        bool `json:"logHeaderNotFound"`
}

func CreateConfig() *Config {
	return &Config{
		ResponseTimeHeaderName: "Tm",
		ResponseTimeLimitMs:    "80",
		CookieSetHeaderValue:   "invalidated",
		PartitionedHeaderValue: "; SameSite=None; Partitioned;",

		EnableCookieInvalidation: true,
		LogStartup:               true,
		LogSetCookie:             true,
		LogLimitNotReached:       true,
		LogHeaderNotFound:        true,
	}
}

type Plugin struct {
	next    http.Handler
	name    string
	config  *Config
	limitMs int
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.LogStartup {
		os.Stderr.WriteString(fmt.Sprintf("RTB :    Init config : %v\n", config))
	}

	limit, err := strconv.Atoi(config.ResponseTimeLimitMs)
	if err != nil {
		return nil, fmt.Errorf("RTB :    cannot parse ResponseTimeLimit, got %v", config.ResponseTimeLimitMs)
	}

	return &Plugin{
		next:    next,
		name:    name,
		config:  config,
		limitMs: limit,
	}, nil
}

func (a *Plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	myWriter := &responseWriter{
		writer:                 rw,
		config:                 a.config,
		ResponseTimeHeaderName: a.config.ResponseTimeHeaderName,
		ResponseTimeLimit:      a.limitMs,
		CookieSetHeaderValue:   a.config.CookieSetHeaderValue,
		PartitionedHeaderValue: a.config.PartitionedHeaderValue,
	}

	podIdHeaderValue := req.Header.Get("pod-id")
	if len(podIdHeaderValue) > 0 {
		req.Header.Set("Cookie", fmt.Sprintf("pod-id=%s", podIdHeaderValue))
		os.Stderr.WriteString(fmt.Sprintf("RTB : updated request header Cookie with the value of %s\n", podIdHeaderValue))
	} else {
		os.Stderr.WriteString("RTB : no pod-id header in the request\n")
	}

	a.next.ServeHTTP(myWriter, req)
}
