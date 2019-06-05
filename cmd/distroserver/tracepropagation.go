package main

import (
	"net/http"

	"github.com/honeycombio/beeline-go/trace"
)

// TracePropagationMiddleware creates a honeycomb trace from a serialized trace in the query parameters
// or creates a new trace and adds the trace parameter to the query for propagation.
func TracePropagationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		ctx, t := trace.NewTrace(req.Context(), req.URL.Query().Get("trace"))
		req.WithContext(ctx)
		span := t.GetRootSpan()

		queryVals := req.URL.Query()
		queryVals.Set("trace", span.SerializeHeaders())
		req.URL.RawQuery = queryVals.Encode()

		next.ServeHTTP(w, req)
	})
}
