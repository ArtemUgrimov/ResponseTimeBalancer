package ResponseTimeBalancer

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type responseWriter struct {
	writer http.ResponseWriter
	config *Config

	ResponseTimeHeaderName string
	ResponseTimeLimit      int
	CookieSetHeaderValue   string
	PartitionedHeaderValue string
}

func (r *responseWriter) Header() http.Header {
	return r.writer.Header()
}

func (r *responseWriter) Write(bytes []byte) (int, error) {
	return r.writer.Write(bytes)
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.invalidateCookie()
	r.enablePartitioned()

	r.writer.WriteHeader(statusCode)
}

func (r *responseWriter) invalidateCookie() {
	if !r.config.EnableCookieInvalidation {
		return
	}

	tmStr := r.writer.Header().Get(r.ResponseTimeHeaderName)
	if len(tmStr) > 0 {
		tm, err := strconv.Atoi(tmStr)
		if err == nil {
			if tm > r.ResponseTimeLimit {
				r.writer.Header().Set("Set-Cookie", r.CookieSetHeaderValue)

				if r.config.LogSetCookie {
					os.Stderr.WriteString(
						fmt.Sprintf(
							"RTL plugin:    Response time = %d. Set-Cookie: %s\n",
							tm,
							r.CookieSetHeaderValue,
						),
					)
				}
			} else {
				if r.config.LogLimitNotReached {
					os.Stderr.WriteString(
						fmt.Sprintf(
							"RTL plugin:    Response time = %d. Limit (%d) is not reached. Skip\n",
							tm,
							r.ResponseTimeLimit,
						),
					)
				}
			}
		}
	}
}

func (r *responseWriter) enablePartitioned() {
	if len(r.config.PartitionedHeaderValue) == 0 {
		return
	}

	// https://github.com/traefik/traefik/issues/10117
	setCookieHeader := r.writer.Header().Get("Set-Cookie")
	if len(setCookieHeader) > 0 {
		if strings.Contains(setCookieHeader, "Partitioned") {
			os.Stderr.WriteString("Set-Cookie already contains Partitioned value. Skip\n")
		} else {
			// add Partitioned value
			r.writer.Header().Set("Set-Cookie", fmt.Sprintf("%s %s", setCookieHeader, r.config.PartitionedHeaderValue))
			os.Stderr.WriteString(fmt.Sprintf("Added %s value to the cookies\n", r.config.PartitionedHeaderValue))
		}
	}
}

func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.writer.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("RTL plugin:    %T is not a http.Hijacker", r.writer)
	}

	return hijacker.Hijack()
}

func (r *responseWriter) Flush() {
	if flusher, ok := r.writer.(http.Flusher); ok {
		flusher.Flush()
	}
}
