package ResponseTimeBalancer

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
)

type responseWriter struct {
	writer http.ResponseWriter
	config *Config

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
	} else {
		if r.config.LogHeaderNotFound {
			os.Stderr.WriteString(
				fmt.Sprintf(
					"RTL plugin:    Could not find header %s\n",
					r.ResponseTimeHeaderName,
				),
			)
		}
	}

	r.writer.WriteHeader(statusCode)
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
