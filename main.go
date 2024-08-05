package ResponseTimeBalancer

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

// Config the plugin configuration.
type Config struct {
	CookieName        string `json:"cookie_name,omitempty"`
	ResponseTimeLimit int64  `json:"response_time_limit_ms,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		CookieName:        "pod-id",
		ResponseTimeLimit: 50,
	}
}

// ResponseTimeLimit a ResponseTimeLimit plugin.
type ResponseTimeLimit struct {
	next              http.Handler
	name              string
	CookieName        string
	ResponseTimeLimit int64
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &ResponseTimeLimit{
		next:              next,
		name:              name,
		CookieName:        config.CookieName,
		ResponseTimeLimit: config.ResponseTimeLimit,
	}, nil
}

func (a *ResponseTimeLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	tsBefore := time.Now().UnixMilli()

	myWriter := &responseWriter{
		writer:            rw,
		CookieName:        a.CookieName,
		ResponseTimeLimit: a.ResponseTimeLimit,
		startTime:         tsBefore,
	}

	os.Stderr.WriteString("\n\n\nBEFORE SERVE\n\n\n")

	a.next.ServeHTTP(myWriter, req)
	// a.next.ServeHTTP(rw, req)

	os.Stderr.WriteString("\n\n\nAFTER SERVE\n\n\n")

	tsAfter := time.Now().UnixMilli()
	os.Stderr.WriteString(fmt.Sprintf("Response time = %d", tsAfter-tsBefore))
	os.Stdout.WriteString(fmt.Sprintf("Response time = %d", tsAfter-tsBefore))
}

type responseWriter struct {
	writer            http.ResponseWriter
	CookieName        string
	ResponseTimeLimit int64

	startTime int64
}

func (r *responseWriter) Header() http.Header {
	return r.writer.Header()
}

func (r *responseWriter) Write(bytes []byte) (int, error) {
	return r.writer.Write(bytes)
}

func (r *responseWriter) WriteHeader(statusCode int) {

	tsAfter := time.Now().UnixMilli()
	os.Stderr.WriteString(fmt.Sprintf("Response time = %d", tsAfter-r.startTime))

	r.writer.Header().Add("Poceluj moju zalupu", "1488")
	// r.writer.Header().Del(r.CookieName)
	// if tsAfter-r.startTime > r.ResponseTimeLimit {
	// 	// Delete set-cookie headers

	// 	os.Stderr.WriteString(fmt.Sprintf("Delete cookies because of response time = %d", tsAfter-r.startTime))
	// }

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
