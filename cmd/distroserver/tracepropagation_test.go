package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/trace"
)

func TestTracePropagationMiddleware(t *testing.T) {
	beeline.Init(beeline.Config{STDOUT: true})

	ctx, tr := trace.NewTrace(context.Background(), "")
	serializedHeader := tr.GetRootSpan().SerializeHeaders()

	u, _ := url.Parse("/foo")
	queryVals := u.Query()
	queryVals.Set("trace", serializedHeader)
	u.RawQuery = queryVals.Encode()

	request := httptest.NewRequest(http.MethodGet, u.String(), nil)
	request.WithContext(ctx)
	wr := httptest.NewRecorder()
	TracePropagationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		propagatedTraceData := r.URL.Query().Get("trace")
		if propagatedTraceData != serializedHeader {
			t.Errorf("wrong serialized header decoded")
			t.Logf("got: %s", propagatedTraceData)
			t.Logf("expected: %s", serializedHeader)
		}
	})).ServeHTTP(wr, request)
}
