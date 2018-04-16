package pipe

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
)

type ResponsePipe interface {
	Transform(*http.Response) error
}

// PipeHandler passes the output of the wrapped Handler through the supplied ResponsePipe.
type PipeHandler struct {
	ResponsePipe ResponsePipe
	Handler      http.Handler
}

func (h *PipeHandler) copyResponse(w http.ResponseWriter, r *http.Response) {
	for header := range map[string][]string(r.Header) {
		w.Header().Set(header, r.Header.Get(header))
	}
	w.WriteHeader(r.StatusCode)
	io.Copy(w, r.Body)
}

func (h *PipeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b := httptest.NewRecorder()

	h.Handler.ServeHTTP(b, r)

	response := b.Result()

	queryValues := r.URL.Query()

	if _, ok := queryValues["raw"]; ok {
		h.copyResponse(w, response)
		return
	}

	err := h.ResponsePipe.Transform(response)
	if err != nil {
		log.Printf("An error ocurred while transforming response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.copyResponse(w, response)
}
