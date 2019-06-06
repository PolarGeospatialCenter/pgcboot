package pipe

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"testing"
)

func TestPipeExec(t *testing.T) {
	p := &PipeExec{Command: []string{"/bin/cat"}, ContentType: "application/text"}
	rec := httptest.NewRecorder()
	rec.Write([]byte("Hello world!"))
	rec.Header().Set("Content-type", "application/json")

	r := rec.Result()
	err := p.Transform(context.Background(), r)
	if err != nil {
		t.Fatalf("An error ocurred while transforming content: %s", err)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("Unable to read body: %v", err)
	}
	if string(body) != "Hello world!" {
		t.Fatalf("The output doesn't match the expected value: %s", body)
	}
	if r.Header.Get("Content-type") != "application/text" {
		t.Fatalf("Content-type not set correctly: %s", r.Header.Get("Content-type"))
	}
}
