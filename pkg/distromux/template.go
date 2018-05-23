package distromux

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/PolarGeospatialCenter/pgcboot/pkg/api"
	"github.com/PolarGeospatialCenter/pgcboot/pkg/handler/pipe"
	"github.com/PolarGeospatialCenter/pgcboot/pkg/handler/template"
)

// TemplateData is the struct that will be passed into the template at render time
type TemplateData struct {
	BaseURL       string
	DistroVars    map[string]interface{}
	RequestParams map[string]interface{}
	RawQuery      string
}

// TemplateRenderer implements the RenderManager interface.
type TemplateRenderer struct {
	DefaultTemplate  string
	FileNameTemplate string
	DistroVars       map[string]interface{}
	DataSources      api.EndpointMap
}

func (tr *TemplateRenderer) getBaseURL(r *http.Request) (string, error) {
	relpath := ""
	if r.URL.Path[0] == '/' {
		pathels := strings.Split(r.URL.Path[1:], "/")
		relpath = strings.Join(pathels[0:len(pathels)-1], "/")
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	host := r.Host

	if relpath != "" {
		u, err := url.Parse(fmt.Sprintf("%s://%s/%s", scheme, host, relpath))
		return u.String(), err
	}
	u, err := url.Parse(fmt.Sprintf("%s://%s", scheme, host))
	return u.String(), err
}

// getNode gets the node associated with this request.
func (tr *TemplateRenderer) getTemplateData(r *http.Request) (*TemplateData, error) {

	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	baseURL, err := tr.getBaseURL(r)
	if err != nil {
		return nil, err
	}

	templateData := &TemplateData{
		RawQuery:   r.URL.RawQuery,
		DistroVars: tr.DistroVars,
		BaseURL:    baseURL,
	}
	requestParams := make(map[string]interface{})
	for key, value := range query {
		switch len(value) {
		case 1:
			requestParams[key] = value[0]
		case 0:
		default:
			requestParams[key] = value
		}
	}
	templateData.RequestParams = requestParams

	return templateData, nil
}

func templateNames(t *template.Template) map[string]string {
	templateList := make(map[string]string)
	log.Printf("Loading template names for: %v", t)
	for _, tmpl := range t.Templates() {
		log.Printf("Found: %s", tmpl.Name())
		templateList[strings.Split(tmpl.Name(), ".")[0]] = tmpl.Name()
	}
	return templateList
}

// TemplateSelector chooses the appropriate template to use for handling the request.
// Search order:
// 1. node.Role match
// 2. DefaultTemplate
func (tr *TemplateRenderer) TemplateSelector(r *http.Request, t *template.Template) (string, error) {
	data, err := tr.getTemplateData(r)
	if err != nil {
		switch err.(type) {
		case templatehandler.ErrNotFound:
			// Specified node not found, return default template
			return tr.DefaultTemplate, nil
		default:
			return "", fmt.Errorf("unexpected error getting template data in template selector: %v", err)
		}
	}

	templateMap := templateNames(t)
	nameTemplate, err := template.New("nametemplate").Parse(tr.FileNameTemplate)
	if err != nil {
		return "", fmt.Errorf("unable to parse template file selection template string: %s", err)
	}
	name := bytes.NewBufferString("")
	err = nameTemplate.Execute(name, data)
	if err != nil {
		return "", fmt.Errorf("unable to render template file selection template string: %s", err)
	}

	if name.String() != "" {
		templateName, ok := templateMap[name.String()]
		if ok {
			return templateName, nil
		}
	}

	log.Printf("Chose template: %s", tr.DefaultTemplate)
	return tr.DefaultTemplate, nil
}

// GetData returns the node data associated with this request, if any.
func (tr *TemplateRenderer) GetData(r *http.Request) (interface{}, error) {
	data, err := tr.getTemplateData(r)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, templatehandler.ErrNotFound{Message: "Unable to find data for request"}
	}
	return data, nil
}

func (tr *TemplateRenderer) TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"api": tr.DataSources.Call,
	}
}

// TemplateEndpoint describes the configuration of an endpoint based on golang
// templates.
type TemplateEndpoint struct {
	TemplatePath     string   `mapstructure:"template_path"`
	ContentType      string   `mapstructure:"content_type"`
	DefaultTemplate  string   `mapstructure:"default_template"`
	FileNameTemplate string   `mapstructure:"filename_template"`
	PostRender       []string `mapstructure:"post_render"`
}

// CreateHandler returns a handler for the endpoint described by this configuration
func (e *TemplateEndpoint) CreateHandler(basepath string, _ string, distroVars map[string]interface{}, dataSources api.EndpointMap) (http.Handler, error) {
	var h http.Handler
	headers := make(map[string]string)
	headers["Content-type"] = e.ContentType
	tr := &TemplateRenderer{DefaultTemplate: e.DefaultTemplate, DistroVars: distroVars, DataSources: dataSources}
	log.Println(tr)
	h, err := templatehandler.NewTemplateHandler(filepath.Join(basepath, e.TemplatePath), headers, tr)
	if err != nil {
		return nil, err
	}

	for _, post := range e.PostRender {
		cmd := strings.Split(post, " ")
		h = &pipe.PipeHandler{ResponsePipe: &pipe.PipeExec{Command: cmd, ContentType: e.ContentType}, Handler: h}
	}

	return h, nil
}
