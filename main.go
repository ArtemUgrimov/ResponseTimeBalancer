package ResponseTimeBalancer

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
)

type Config struct {
	CookiePodIdName        string `json:"cookiePodIdName"`
	AdditionalCookie       string `json:"additionalCookie"`
	ResponseTimeHeaderName string `json:"responseTimeHeaderName"`
	ResponseTimeDiffMs     string `json:"responseTimeDiffMsToInvalidate"`

	LogStart             bool `json:"logStart"`
	LogCookieFound       bool `json:"logCookieFound"`
	LogCookieResetToBest bool `json:"logCookieResetToBest"`
	LogIdle              bool `json:"logIdle"`
	LogBestUpdate        bool `json:"logBestUpdate"`
}

func CreateConfig() *Config {
	return &Config{
		CookiePodIdName:        "pod-id",
		AdditionalCookie:       "",
		ResponseTimeHeaderName: "Tm",
		ResponseTimeDiffMs:     "50",

		LogStart:             true,
		LogCookieFound:       true,
		LogCookieResetToBest: true,
		LogIdle:              true,
		LogBestUpdate:        true,
	}
}

type CurrentBestPod struct {
	tm    int
	podId string
}

type Plugin struct {
	next http.Handler
	name string

	config      *Config
	diffMs      int // it is declared here to avoid conversion str->int on each request
	currentBest CurrentBestPod
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.LogStart {
		os.Stderr.WriteString(fmt.Sprintf("RTL plugin:   init config : %v\n", config))
	}

	diffMs, err := strconv.Atoi(config.ResponseTimeDiffMs)
	if err != nil {
		return nil, fmt.Errorf("RTL plugin:   cannot parse ResponseTimeDiffMs, got %v", config.ResponseTimeDiffMs)
	}

	return &Plugin{
		next:   next,
		name:   name,
		config: config,
		diffMs: diffMs,

		currentBest: CurrentBestPod{
			tm:    9999,
			podId: "",
		},
	}, nil
}

func (a *Plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	myWriter := &responseWriter{
		writer: rw,
		plugin: a,
	}
	cookie, err := req.Cookie(a.config.CookiePodIdName)
	if err == nil {
		myWriter.currentPodIdName = cookie.Value
		if a.config.LogCookieFound {
			os.Stderr.WriteString(fmt.Sprintf("RTL plugin:   cookie %s found : %s\n", a.config.CookiePodIdName, cookie.Value))
		}
	} else {
		if a.config.LogCookieFound {
			os.Stderr.WriteString(fmt.Sprintf("RTL plugin:   cookie %s not found\n", a.config.CookiePodIdName))
		}
	}

	a.next.ServeHTTP(myWriter, req)
}
