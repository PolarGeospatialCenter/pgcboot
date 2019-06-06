package pipe

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type loopbackHandler struct{}

func (l *loopbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/plain")
	io.Copy(w, r.Body)
}

type testTransformer struct{}

func (t *testTransformer) Transform(_ context.Context, r *http.Response) error {
	r.Header.Set("Content-type", "application/json")
	currentBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	newBodyData := &struct {
		Body string
	}{
		Body: string(currentBody),
	}

	newBodyBytes, err := json.Marshal(newBodyData)
	if err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(newBodyBytes))
	return nil
}

func TestPipeHandler(t *testing.T) {
	body := bytes.NewBufferString("Hello world!")
	request, err := http.NewRequest("POST", "/transformer", body)
	if err != nil {
		t.Fatalf("Unable to create sample request: %v", err)
	}

	h := &PipeHandler{ResponsePipe: &testTransformer{}, Handler: &loopbackHandler{}}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request)
	response := w.Result()

	if response.StatusCode != http.StatusOK {
		t.Errorf("Wrong status returned: %d", response.StatusCode)
	}

	if response.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Header not transformed properly: %s", response.Header.Get("Content-Type"))
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Unable to read transformed response body: %s", err)
	}

	if string(responseBody) != `{"Body":"Hello world!"}` {
		t.Errorf("Transormed body contains the wrong contents: '%s'", responseBody)
	}

}

func TestPipeHandlerRaw(t *testing.T) {
	body := bytes.NewBufferString("Hello world!")
	request, err := http.NewRequest("POST", "/transformer?raw", body)
	if err != nil {
		t.Fatalf("Unable to create sample request: %v", err)
	}

	h := &PipeHandler{ResponsePipe: &testTransformer{}, Handler: &loopbackHandler{}}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request)
	response := w.Result()

	if response.StatusCode != http.StatusOK {
		t.Errorf("Wrong status returned: %d", response.StatusCode)
	}

	if response.Header.Get("Content-Type") != "text/plain" {
		t.Errorf("Header not transformed properly: %s", response.Header.Get("Content-Type"))
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Unable to read transformed response body: %s", err)
	}

	if string(responseBody) != `Hello world!` {
		t.Errorf("Transormed body contains the wrong contents: '%s'", responseBody)
	}

}
