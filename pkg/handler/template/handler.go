package templatehandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
)

// ErrNotFound should be returned when a template could not be found to serve the request.
type ErrNotFound struct {
	Message string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s", e.Message)
}

// RenderManagers are responsible for choosing the correct template to render and what data to populate it with.  Embed
// the DefaultRenderManager for basic functionality.
type RenderManager interface {
	GetData(*http.Request) (interface{}, error)
	TemplateSelector(*http.Request, *template.Template) (string, error)
	TemplateFuncs() template.FuncMap
}

// DefaultRenderManager is an implementation of the RenderManager interface that selects the first template available
// and populates it with whatever data is assigned to the Data element of the DefaultRenderManager.
type DefaultRenderManager struct {
	Data interface{}
}

// Returns the name of the first template from all those attached to t.
func (m *DefaultRenderManager) TemplateSelector(r *http.Request, t *template.Template) (string, error) {
	return t.Templates()[0].Name(), nil
}

// GetData returns the Data element of the DefaultRenderManager instance.
func (m *DefaultRenderManager) GetData(r *http.Request) (interface{}, error) {
	return m.Data, nil
}

// TemplateHandler implements the http.Handler interface.  Typically this would be used to implement a very simple api
// using go templates and a custom RenderManager.
type TemplateHandler struct {
	templatePath string
	Template     *template.Template
	Headers      map[string]string
	RenderManager
}

// NewTemplateHandler returns a TemplateHandler with all templates found under the provided path loaded into Template.
// The supplied RenderManager and Headers are also populated.
func NewTemplateHandler(path string, headers map[string]string, rm RenderManager) (*TemplateHandler, error) {
	th := &TemplateHandler{RenderManager: rm, Headers: headers, templatePath: path}
	err := th.ReloadTemplates()
	return th, err
}

// ReloadTemplates walks the supplied template path loading all templates into the handler
func (th *TemplateHandler) ReloadTemplates() error {
	templateFiles := make([]string, 0)
	err := filepath.Walk(th.templatePath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			templateFiles = append(templateFiles, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	t := template.New("templatebase").Funcs(th.RenderManager.TemplateFuncs())

	t, err = t.ParseFiles(templateFiles...)
	if err != nil {
		return err
	}

	th.Template = t
	return nil
}

// Uses embedded RenderManager to render the template that's appropriate for the request.
// The output of the rendered template is written to the supplied io.Writer.
func (t *TemplateHandler) renderTemplate(w io.Writer, r *http.Request) error {
	// Filter data
	data, err := t.GetData(r)
	if err != nil {
		return err
	}

	// Select the template to render
	template_name, err := t.TemplateSelector(r, t.Template)
	if err != nil {
		return err
	}

	return t.Template.ExecuteTemplate(w, template_name, data)
}

// RenderJsonError writes the provided err to the ResponseWriter in JSON format.
func RenderJsonError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(status)
	err_body := make(map[string]string)
	err_body["msg"] = fmt.Sprintf("%v", err)
	enc := json.NewEncoder(w)
	enc.Encode(&err_body)
}

// ServeHTTP handles requests from the user.  Any error raised during the process other than ErrTemplateNotFound results
// in a 500 status being returned to the user and a more detailed log being written.
func (t *TemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body bytes.Buffer
	// render template
	err := t.renderTemplate(&body, r)
	if _, ok := err.(ErrNotFound); ok {
		RenderJsonError(w, http.StatusNotFound, err)
		log.Printf("Not Found: %s", err)
		return
	} else if err != nil {
		RenderJsonError(w, http.StatusInternalServerError, fmt.Errorf("Internal server error. Please consult the server logs."))
		log.Printf("An error ocurred while handling %v: %s", r, err)
		return
	}
	for header, value := range t.Headers {
		w.Header().Set(header, value)
	}

	if _, err := io.Copy(w, &body); err != nil {
		log.Printf("Unable to write body to client: %s", err)
	}
}
