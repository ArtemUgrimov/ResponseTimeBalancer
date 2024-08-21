package ResponseTimeBalancer

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
)

// Config the plugin configuration.
type Config struct {
	ResponseTimeHeaderName string `json:"responseTimeHeaderName"`
	ResponseTimeLimitMs    string `json:"responseTimeLimitMs"`
	CookieSetHeaderValue   string `json:"cookieSetHeaderValue"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		ResponseTimeHeaderName: "Tm",
		ResponseTimeLimitMs:    "50",
		CookieSetHeaderValue:   "invalidated",
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
	os.Stderr.WriteString(fmt.Sprintf("ResponseTimeLimit plugin:    Init config : %v\n", config))

	limit, err := strconv.Atoi(config.ResponseTimeLimitMs)
	if err != nil {
		return nil, fmt.Errorf("cannot parse ResponseTimeLimit, got %v", config.ResponseTimeLimitMs)
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

type responseWriter struct {
	writer                 http.ResponseWriter
	ResponseTimeHeaderName string
	ResponseTimeLimit      int
	CookieSetHeaderValue   string
}

func (r *responseWriter) Header() http.Header {
	return r.writer.Header()
}

func (r *responseWriter) Write(bytes []byte) (int, error) {
	return r.writer.Write(bytes)
}

func (r *responseWriter) WriteHeader(statusCode int) {
	tmStr := r.writer.Header().Get(r.ResponseTimeHeaderName)
	if len(tmStr) > 0 {
		tm, err := strconv.Atoi(tmStr)
		if err == nil {
			if tm > r.ResponseTimeLimit {
				r.writer.Header().Set("Set-Cookie", r.CookieSetHeaderValue)
				os.Stderr.WriteString(
					fmt.Sprintf(
						"ResponseTimeLimit plugin:   Response time = %d. Set-Cookie: %s\n", tm, r.CookieSetHeaderValue))
			} else {
				os.Stderr.WriteString(
					fmt.Sprintf(
						"ResponseTimeLimit plugin:   Response time = %d. Limit (%d) is not reached. Skip\n", tm, r.ResponseTimeLimit))
			}
		}
	} else {
		os.Stderr.WriteString(fmt.Sprintf("ResponseTimeLimit plugin:   Could not find header %s\n", r.ResponseTimeHeaderName))
	}

	r.writer.WriteHeader(statusCode)
}

func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.writer.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T is not a http.Hijacker", r.writer)
	}

	return hijacker.Hijack()
}

func (r *responseWriter) Flush() {
	if flusher, ok := r.writer.(http.Flusher); ok {
		flusher.Flush()
	}
}
