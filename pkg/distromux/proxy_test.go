package distromux

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestProxyHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(wr http.ResponseWriter, r *http.Request) {
		wr.WriteHeader(http.StatusOK)
		wr.Write([]byte("Test Text."))
	})
	s := &http.Server{
		Addr:         "127.0.0.1:43210",
		Handler:      mux,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	go s.ListenAndServe()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer s.Shutdown(ctx)

	testTarget := fmt.Sprintf("http://%s", s.Addr)
	endpoint := &ProxyEndpoint{TargetURL: testTarget}
	h, err := endpoint.CreateHandler("", "/local/", nil)
	if err != nil {
		t.Fatalf("Unable to create handler: %v", err)
	}

	response := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "/local/", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}

	h.ServeHTTP(response, request)
	body, err := ioutil.ReadAll(response.Result().Body)
	if err != nil {
		t.Fatalf("Unable to read body: %v", err)
	}

	directRequest, err := http.NewRequest("GET", testTarget, nil)
	if err != nil {
		t.Fatalf("Unable to create direct request: %v", err)
	}

	directResponse, err := http.DefaultTransport.RoundTrip(directRequest)
	if err != nil {
		t.Fatalf("Unable to make direct request: %v", err)
	}

	directBody, err := ioutil.ReadAll(directResponse.Body)
	if err != nil {
		t.Fatalf("Unable to read direct response body: %v", err)
	}

	if response.Result().StatusCode != directResponse.StatusCode {
		t.Errorf("Proxy returned incorrect status: %d != %d", response.Result().StatusCode, directResponse.StatusCode)
	}

	if string(directBody) != string(body) {
		diff := diffmatchpatch.New()
		d := diff.DiffMain(string(directBody), string(body), false)
		t.Logf("Proxied response: %s", body)
		t.Logf("Diff: %s", diff.DiffPrettyText(d))
		t.Errorf("Direct request body doesn't match body of proxied return.")
	}
}
