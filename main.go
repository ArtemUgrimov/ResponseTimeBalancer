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

	LogStartup         bool `json:"logStartup"`
	LogSetCookie       bool `json:"logSetCookie"`
	LogLimitNotReached bool `json:"logLimitNotReached"`
	LogHeaderNotFound  bool `json:"logHeaderNotFound"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		ResponseTimeHeaderName: "Tm",
		ResponseTimeLimitMs:    "80",
		CookieSetHeaderValue:   "invalidated",

		LogStartup:         true,
		LogSetCookie:       true,
		LogLimitNotReached: true,
		LogHeaderNotFound:  true,
	}
}

// ResponseTimeLimit a ResponseTimeLimit plugin.
type ResponseTimeLimit struct {
	next    http.Handler
	name    string
	config  *Config
	limitMs int
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.LogStartup {
		os.Stderr.WriteString(fmt.Sprintf("RTL plugin:    Init config : %v\n", config))
	}

	limit, err := strconv.Atoi(config.ResponseTimeLimitMs)
	if err != nil {
		return nil, fmt.Errorf("RTL plugin:    cannot parse ResponseTimeLimit, got %v", config.ResponseTimeLimitMs)
	}

	return &ResponseTimeLimit{
		next:    next,
		name:    name,
		config:  config,
		limitMs: limit,
	}, nil
}

func (a *ResponseTimeLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	myWriter := &responseWriter{
		writer:                 rw,
		ResponseTimeHeaderName: a.config.ResponseTimeHeaderName,
		ResponseTimeLimit:      a.limitMs,
		CookieSetHeaderValue:   a.config.CookieSetHeaderValue,
	}

	a.next.ServeHTTP(myWriter, req)
}
