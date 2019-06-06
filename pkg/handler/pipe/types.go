package pipe

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/honeycombio/beeline-go/trace"
)

type ResponsePipe interface {
	Transform(*http.Response) error
}

// PipeHandler passes the output of the wrapped Handler through the supplied ResponsePipe.
type PipeHandler struct {
	ResponsePipe ResponsePipe
	Handler      http.Handler
}

func (h *PipeHandler) copyResponse(w http.ResponseWriter, r *http.Response) (int64, error) {
	for header := range map[string][]string(r.Header) {
		w.Header().Set(header, r.Header.Get(header))
	}
	w.WriteHeader(r.StatusCode)
	count, err := io.Copy(w, r.Body)
	return count, err
}

func (h *PipeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parentSpan := trace.GetSpanFromContext(r.Context())
	var span *trace.Span
	if parentSpan != nil {
		var ctx context.Context
		ctx, span = parentSpan.CreateChild(r.Context())
		r = r.WithContext(ctx)
		span.AddField("name", "PipeHandler")
	}
	b := httptest.NewRecorder()

	h.Handler.ServeHTTP(b, r)

	response := b.Result()
	queryValues := r.URL.Query()

	if _, ok := queryValues["raw"]; ok {
		_, err := h.copyResponse(w, response)
		if err != nil {
			log.Printf("error replaying raw response: %v", err)
		}
		return
	}

	err := h.ResponsePipe.Transform(response)
	if err != nil {
		log.Printf("An error ocurred while transforming response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	count, err := h.copyResponse(w, response)
	if err != nil {
		log.Printf("error replaying transformed response: %v", err)
	}

	if span != nil {
		span.AddField("response.transformed_length", count)
	}

}
