package distromux

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/PolarGeospatialCenter/pgcboot/pkg/api"
	"github.com/PolarGeospatialCenter/pgcboot/pkg/handler/pipe"
	"github.com/PolarGeospatialCenter/pgcboot/pkg/handler/template"
)

// TemplateData is the struct that will be passed into the template at render time
type TemplateData struct {
	BaseURL       string
	DistroVars    DistroVars
	RequestParams map[string]string
	RawQuery      string
}

// TemplateRenderer implements the RenderManager interface.
type TemplateRenderer struct {
	DefaultTemplate  string
	FileNameTemplate string
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

	distroVars, ok := DistroVarsFromContext(r.Context())
	if !ok {
		return nil, fmt.Errorf("unable to read distro vars from context")
	}

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
		DistroVars: distroVars,
		BaseURL:    baseURL,
	}
	requestParams := make(map[string]string)
	for key, value := range query {
		switch len(value) {
		case 1:
			requestParams[key] = value[0]
		case 0:
		default:
			requestParams[key] = strings.Join(value, ",")
		}
	}
	templateData.RequestParams = requestParams

	return templateData, nil
}

// TemplateSelector chooses the appropriate template to use for handling the request.
// Search order:
// 1. node.Role match
// 2. DefaultTemplate
func (tr *TemplateRenderer) TemplateSelector(r *http.Request, t *template.Template) (string, error) {
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
		"api":  tr.DataSources.Call,
		"join": TemplateJoinWrapper,
	}
}

func convertInterfaceToString(item interface{}) (string, error) {
	switch item.(type) {
	case string:
		return item.(string), nil
	case int:
		return strconv.Itoa(item.(int)), nil
	default:
		return "", fmt.Errorf("Join item could not be converted from %T to string: %v", item, item)
	}
}

func TemplateJoinWrapper(data interface{}, sep string) (string, error) {
	var stringValues []string
	var getItem func(reflect.Value, int) reflect.Value
	s := reflect.ValueOf(data)
	switch reflect.TypeOf(data).Kind() {
	case reflect.Map:
		keys := s.MapKeys()
		getItem = func(value reflect.Value, idx int) reflect.Value {
			return value.MapIndex(keys[idx])
		}
	case reflect.Slice:
		getItem = func(value reflect.Value, idx int) reflect.Value {
			return value.Index(idx)
		}
	default:
		return "", fmt.Errorf("Join unsupported for type: %T", data)
	}
	var err error
	stringValues = make([]string, s.Len())
	for i := 0; i < s.Len(); i++ {
		item := getItem(s, i).Interface()
		stringValues[i], err = convertInterfaceToString(item)
		if err != nil {
			break
		}
	}
	return strings.Join(stringValues, sep), err
}

// TemplateEndpoint describes the configuration of an endpoint based on golang
// templates.
type TemplateEndpoint struct {
	TemplatePath     string   `mapstructure:"template_path"`
	ContentType      string   `mapstructure:"content_type"`
	DefaultTemplate  string   `mapstructure:"default_template"`
	PostRender       []string `mapstructure:"post_render"`
}

// CreateHandler returns a handler for the endpoint described by this configuration
func (e *TemplateEndpoint) CreateHandler(basepath string, _ string, dataSources api.EndpointMap) (http.Handler, error) {
	var h http.Handler
	headers := make(map[string]string)
	headers["Content-type"] = e.ContentType
	tr := &TemplateRenderer{DefaultTemplate: e.DefaultTemplate, DataSources: dataSources}
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
