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
	CookieName             string `json:"cookie_name,omitempty"`
	ResponseTimeHeaderName string `json:"response_time_header_name,omitempty"`
	ResponseTimeLimit      int    `json:"response_time_limit_ms,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		CookieName:             "pod-id",
		ResponseTimeHeaderName: "Tm",
		ResponseTimeLimit:      50,
	}
}

// ResponseTimeLimit a ResponseTimeLimit plugin.
type ResponseTimeLimit struct {
	next                   http.Handler
	name                   string
	CookieName             string
	ResponseTimeHeaderName string
	ResponseTimeLimit      int
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	os.Stderr.WriteString(fmt.Sprintf("Creating a ResponseTimeLimit plugin. Config : %v\n", config))

	return &ResponseTimeLimit{
		next:                   next,
		name:                   name,
		CookieName:             config.CookieName,
		ResponseTimeHeaderName: config.ResponseTimeHeaderName,
		ResponseTimeLimit:      config.ResponseTimeLimit,
	}, nil
}

func (a *ResponseTimeLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	myWriter := &responseWriter{
		writer:                 rw,
		CookieName:             a.CookieName,
		ResponseTimeHeaderName: a.ResponseTimeHeaderName,
		ResponseTimeLimit:      a.ResponseTimeLimit,
	}

	a.next.ServeHTTP(myWriter, req)
}

type responseWriter struct {
	writer                 http.ResponseWriter
	CookieName             string
	ResponseTimeHeaderName string
	ResponseTimeLimit      int
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
		os.Stderr.WriteString(fmt.Sprintf("Response time = %s\n", tmStr))
		tm, err := strconv.Atoi(tmStr)
		if err == nil {
			if tm > r.ResponseTimeLimit {
				r.writer.Header().Set("Set-Cookie", fmt.Sprintf("%s=invalidated", r.CookieName))
				os.Stderr.WriteString(fmt.Sprintf("Deleting cookie with name %s\n", r.CookieName))
			} else {
				os.Stderr.WriteString(fmt.Sprintf("Limit (%d) is not reached. Skip\n", r.ResponseTimeLimit))
			}
		}
	} else {
		os.Stderr.WriteString(fmt.Sprintf("Could not find header %s", r.ResponseTimeHeaderName))
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
