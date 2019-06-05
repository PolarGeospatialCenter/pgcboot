package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/propagation"
	"github.com/honeycombio/beeline-go/trace"
	"github.com/honeycombio/beeline-go/wrappers/hnygorilla"
)

func TestTracePropagationMiddleware(t *testing.T) {
	beeline.Init(beeline.Config{STDOUT: true})

	ctx, tr := trace.NewTrace(context.Background(), "")
	serializedHeader := tr.GetRootSpan().SerializeHeaders()
	expectedTraceData, _ := propagation.UnmarshalTraceContext(serializedHeader)
	defer tr.Send()

	u, _ := url.Parse("/foo")
	queryVals := u.Query()
	queryVals.Set("trace", serializedHeader)
	u.RawQuery = queryVals.Encode()

	request := httptest.NewRequest(http.MethodGet, u.String(), nil)
	request.WithContext(ctx)
	wr := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.GetSpanFromContext(r.Context())
		propagatedTraceData, _ := propagation.UnmarshalTraceContext(span.SerializeHeaders())
		if propagatedTraceData.TraceID != expectedTraceData.TraceID {
			t.Errorf("Trace ID didn't match expected value")
			t.Logf("got: %s", propagatedTraceData.TraceID)
			t.Logf("expected: %s", expectedTraceData.TraceID)
		}
	})

	r := mux.NewRouter()
	r.Use(TracePropagationMiddleware)
	r.Use(hnygorilla.Middleware)
	r.Handle("/foo", testHandler)
	r.ServeHTTP(wr, request)

}
