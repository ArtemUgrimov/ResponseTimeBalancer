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
	writer           http.ResponseWriter
	plugin           *Plugin
	currentPodIdName string
}

func (r *responseWriter) Header() http.Header {
	return r.writer.Header()
}

func (r *responseWriter) Write(bytes []byte) (int, error) {
	return r.writer.Write(bytes)
}

func (r *responseWriter) WriteHeader(statusCode int) {
	tmStr := r.writer.Header().Get(r.plugin.config.ResponseTimeHeaderName)
	if len(tmStr) > 0 {
		tm, err := strconv.Atoi(tmStr)
		if err == nil {
			diff := absDiffInt(tm, r.plugin.currentBest.tm)

			if diff > r.plugin.diffMs && len(r.plugin.currentBest.podId) > 0 {
				// if a diff between current response time and stored response time
				// is greated than the trueshold
				// we set current cookie to the best one that is stored
				cookieStr := fmt.Sprintf(
					"%s=%s%s",
					r.plugin.config.CookiePodIdName,
					r.plugin.currentBest.podId,
					r.plugin.config.AdditionalCookie,
				)
				r.writer.Header().Set("Set-Cookie", cookieStr)

				if r.plugin.config.LogCookieResetToBest {
					os.Stderr.WriteString(
						fmt.Sprintf("RTL plugin:   diff = %d. Set-Cookie: %s\n", diff, cookieStr))
				}
			} else if len(r.currentPodIdName) > 0 {
				// if the diff is smaller than current best
				// we update best with the response time value and new pod-id
				// only if we had pod-id cookie in the request

				if r.currentPodIdName == r.plugin.currentBest.podId {
					// we update timing for the current best pod always
					if r.plugin.config.LogBestUpdate {
						os.Stderr.WriteString(
							fmt.Sprintf(
								"RTL plugin:   updating current best (%s) with tm=%d, old=%d\n",
								r.currentPodIdName,
								tm,
								r.plugin.currentBest.tm,
							),
						)
					}
					r.plugin.currentBest.podId = r.currentPodIdName
					r.plugin.currentBest.tm = tm

				} else if tm < r.plugin.currentBest.tm {
					// but only if the timing is greater on case of another one
					r.plugin.currentBest.podId = r.currentPodIdName
					r.plugin.currentBest.tm = tm

					if r.plugin.config.LogBestUpdate {
						os.Stderr.WriteString(
							fmt.Sprintf(
								"RTL plugin:   updating best pod with %s (tm=%d, diff=%d)\n",
								r.currentPodIdName,
								tm,
								diff,
							),
						)
					}
				}
			} else {
				if r.plugin.config.LogIdle {
					os.Stderr.WriteString(
						fmt.Sprintf(
							"RTL plugin:   response time = %d. diff (%d). skip\n",
							tm,
							diff,
						),
					)
				}
			}
		} else {
			os.Stderr.WriteString(fmt.Sprintf("%s\n", err.Error()))
		}
	} else {
		os.Stderr.WriteString(fmt.Sprintf("RTL plugin:   could not find header %s\n", r.plugin.config.ResponseTimeHeaderName))
	}

	r.writer.WriteHeader(statusCode)
}

func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.writer.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("RTL plugin:   %T is not a http.Hijacker", r.writer)
	}

	return hijacker.Hijack()
}

func (r *responseWriter) Flush() {
	if flusher, ok := r.writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func absDiffInt(x, y int) int {
	if x < y {
		return y - x
	}
	return x - y
}
