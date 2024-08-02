package ResponseTimeBalancer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtemUgrimov/ResponseTimeBalancer"
)

func TestDemo(t *testing.T) {
	cfg := ResponseTimeBalancer.CreateConfig()

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := ResponseTimeBalancer.New(ctx, next, cfg, "ResponseTimeBalancer-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{
		Name:  "pod-id",
		Value: "111",
	})

	handler.ServeHTTP(recorder, req)

	assertCookie(t, req)
}

func assertCookie(t *testing.T, req *http.Request) {
	t.Helper()

	_, err := req.Cookie("pod-id")
	if err != nil {
		t.Errorf("pod-id cookie is present")
	}
}
