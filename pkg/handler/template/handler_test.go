package templatehandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"
)

type TestRenderManager struct {
	TemplateName string
	DefaultRenderManager
}

func (m *TestRenderManager) TemplateSelector(r *http.Request, t *template.Template) (string, error) {
	for _, tmp := range t.Templates() {
		if m.TemplateName == tmp.Name() {
			return m.TemplateName, nil
		}
	}
	return "", ErrNotFound{m.TemplateName}
}

func (m *TestRenderManager) GetData(r *http.Request) (interface{}, error) {
	type SampleData struct {
		Count    int
		Material string
	}
	return SampleData{Count: 2, Material: "wool"}, nil
}

func (m *TestRenderManager) TemplateFuncs() template.FuncMap {
	return template.FuncMap{"hello": func(name string) string { return fmt.Sprintf("Hello %s!", name) }}
}

func sampleTemplateHandler(template_name string) (*TemplateHandler, error) {
	rm := &TestRenderManager{TemplateName: template_name}

	tmpl, err := template.New("incorrect").Funcs(rm.TemplateFuncs()).Parse("This is the wrong template. {{.Count}}")
	if err != nil {
		return nil, err
	}
	_, err = tmpl.New("error").Parse("{\"msg\":\"{{.NonExistData}} items are made of {{.Material}}\"}")
	if err != nil {
		return nil, err
	}
	_, err = tmpl.New("test").Parse("{\"msg\":\"{{.Count}} items are made of {{.Material}}\"}")
	if err != nil {
		return nil, err
	}
	_, err = tmpl.New("hello").Parse("{\"msg\":\"{{ hello .Material }}\"}")
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string)
	headers["Content-type"] = "application/json"
	h := &TemplateHandler{Template: tmpl, RenderManager: rm, Headers: headers}

	return h, nil
}

func TestTemplateRendering(t *testing.T) {
	h, err := sampleTemplateHandler("test")
	if err != nil {
		t.Fatalf("Unable to create template for testing: %v", err)
	}

	var b bytes.Buffer
	err = h.renderTemplate(&b, &http.Request{})
	if err != nil {
		t.Fatalf("Error rendering template: %s", err)
	}

	if b.String() != "{\"msg\":\"2 items are made of wool\"}" {
		t.Fatalf("Wrong string rendered as output: %s", b.String())
	}
}

func TestTemplateFunction(t *testing.T) {
	h, err := sampleTemplateHandler("hello")
	if err != nil {
		t.Fatalf("Unable to create template for testing: %v", err)
	}

	var b bytes.Buffer
	err = h.renderTemplate(&b, &http.Request{})
	if err != nil {
		t.Fatalf("Error rendering template: %s", err)
	}

	if b.String() != "{\"msg\":\"Hello wool!\"}" {
		t.Fatalf("Wrong string rendered as output: %s", b.String())
	}
}

func TestRenderJsonError(t *testing.T) {
	w := httptest.NewRecorder()
	RenderJsonError(w, http.StatusTeapot, fmt.Errorf("I'm a teapot"))

	if w.Result().StatusCode != http.StatusTeapot {
		t.Errorf("Error incorrectly claims not to be a teapot.")
	}
	if w.Result().Header.Get("Content-Type") != "application/json" {
		t.Errorf("Wrong content type returned: %v", w.Result().Header.Get("Content-Type"))
	}

	dec := json.NewDecoder(w.Result().Body)
	body := make(map[string]string)
	dec.Decode(&body)

	if body["msg"] != "I'm a teapot" {
		t.Errorf("Wrong body returned:\n%s", body["msg"])
	}
}

func TestServeHTTPOK(t *testing.T) {
	h, err := sampleTemplateHandler("test")
	if err != nil {
		t.Fatalf("Unable to create template for testing: %v", err)
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "http://localhost/foo", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Unable to create request: %s", err)
	}
	h.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Incorrect status code set: %d", w.Result().StatusCode)
	}

	if w.Result().Header.Get("Content-Type") != "application/json" {
		t.Errorf("Wrong content type set: %s", w.Result().Header.Get("Content-Type"))
	}

	if w.Body.String() != "{\"msg\":\"2 items are made of wool\"}" {
		t.Errorf("Wrong body returned:\n%s", w.Body)
	}
}

func TestServeHTTPNotFound(t *testing.T) {
	h, err := sampleTemplateHandler("NonExist")
	if err != nil {
		t.Fatalf("Unable to create template for testing: %v", err)
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "http://localhost/foo", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Unable to create request: %s", err)
	}
	h.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("Incorrect status code set: %d", w.Result().StatusCode)
	}

	if w.Result().Header.Get("Content-Type") != "application/json" {
		t.Errorf("Wrong content type set: %s", w.Result().Header.Get("Content-Type"))
	}

	dec := json.NewDecoder(w.Result().Body)
	body := make(map[string]string)
	dec.Decode(&body)

	if body["msg"] != "NonExist" {
		t.Errorf("Wrong body returned:\n%s", body["msg"])
	}
}

func TestServeHTTPServerError(t *testing.T) {
	h, err := sampleTemplateHandler("error")
	if err != nil {
		t.Fatalf("Unable to create template for testing: %v", err)
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "http://localhost/foo", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Unable to create request: %s", err)
	}
	h.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("Incorrect status code set: %d", w.Result().StatusCode)
	}

	if w.Result().Header.Get("Content-Type") != "application/json" {
		t.Errorf("Wrong content type set: %s", w.Result().Header.Get("Content-Type"))
	}

	dec := json.NewDecoder(w.Result().Body)
	body := make(map[string]string)
	dec.Decode(&body)

	if body["msg"] != "Internal server error. Please consult the server logs." {
		t.Errorf("Wrong body returned:\n%s", body["msg"])
	}
}
