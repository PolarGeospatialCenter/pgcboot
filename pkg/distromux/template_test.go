package distromux

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"text/template"

	"github.com/PolarGeospatialCenter/pgcboot/pkg/api"
	gock "gopkg.in/h2non/gock.v1"
)

func TestTemplateData(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:8080/branch/master/foo?role=worker", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}

	testVars := DistroVars{}
	testVars["kube_version"] = "1.9.0"
	r = testVars.SetContextForRequest(r)

	renderer := &TemplateRenderer{}

	rawData, err := renderer.GetData(r)
	if err != nil {
		t.Errorf("unable to get data from renderer: %v", err)
	}
	data, ok := rawData.(*TemplateData)
	if !ok {
		t.Errorf("got unexpected data type from renderer: %T", rawData)
	}

	if data.DistroVars == nil || data.DistroVars["kube_version"] != "1.9.0" {
		t.Errorf("got bad value for distrovars: %v", data.DistroVars)
	}

	if data.RequestParams == nil || data.RequestParams["role"] != "worker" {
		t.Errorf("got bad request parameter values %v", data.RequestParams)
	}

	if data.BaseURL != "http://localhost:8080/branch/master" {
		t.Errorf("got bad base url: %v", data.BaseURL)
	}

	if data.RawQuery != "role=worker" {
		t.Errorf("got bad raw query value: %v", data.RawQuery)
	}
}

func TestTemplateAPICall(t *testing.T) {
	gock.DisableNetworking()
	defer gock.EnableNetworking()
	defer gock.Off() // Flush pending mocks after test execution

	e := api.EndpointMap{"test": &api.Endpoint{URL: "https://api.local/v1/foo", Method: http.MethodGet}}
	gock.New("https://api.local/v1").
		Get("/foo").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	renderer := &TemplateRenderer{DataSources: e}
	tmpl, err := template.New("testTemplate").Funcs(renderer.TemplateFuncs()).Parse(`{{ $test := api "test" "" "" "" }}{{ $test.Data.foo }}`)
	if err != nil {
		t.Errorf("unable to create template for testing: %v", err)
	}

	output := bytes.NewBufferString("")
	err = tmpl.Execute(output, nil)
	if err != nil {
		t.Errorf("unable tor render template: %v", err)
	}

	if output.String() != "bar" {
		t.Errorf("unexpected result: expected 'bar' got '%s'", output.String())
	}

}

func TestTemplateJoinFunctionStringSlice(t *testing.T) {
	renderer := &TemplateRenderer{DataSources: api.EndpointMap{}}
	tmpl, err := template.New("templatebase").Funcs(renderer.TemplateFuncs()).Parse(`{{ join .slice "," }}`)
	if err != nil {
		t.Errorf("Unable to parse template for testing: %v", err)
	}
	wr := bytes.NewBufferString("")
	err = tmpl.Execute(wr, map[string]interface{}{"slice": []string{"a", "b", "c"}})
	if err != nil {
		t.Errorf("Error while rendering template: %v", err)
	}

	if wr.String() != "a,b,c" {
		t.Errorf("Unexpected result returned from template renderer: '%s'", wr.String())
	}
}

func TestTemplateJoinFunctionInterfaceSlice(t *testing.T) {
	renderer := &TemplateRenderer{DataSources: api.EndpointMap{}}
	tmpl, err := template.New("templatebase").Funcs(renderer.TemplateFuncs()).Parse(`{{ join .slice "," }}`)
	if err != nil {
		t.Errorf("Unable to parse template for testing: %v", err)
	}
	wr := bytes.NewBufferString("")
	err = tmpl.Execute(wr, map[string]interface{}{"slice": []interface{}{1, "b", "c"}})
	if err != nil {
		t.Errorf("Error while rendering template: %v", err)
	}

	if wr.String() != "1,b,c" {
		t.Errorf("Unexpected result returned from template renderer: '%s'", wr.String())
	}
}

func TestTemplateJoinFunctionMap(t *testing.T) {
	renderer := &TemplateRenderer{DataSources: api.EndpointMap{}}
	tmpl, err := template.New("templatebase").Funcs(renderer.TemplateFuncs()).Parse(`{{ join .map "," }}`)
	if err != nil {
		t.Errorf("Unable to parse template for testing: %v", err)
	}
	wr := bytes.NewBufferString("")
	err = tmpl.Execute(wr, map[string]interface{}{"map": map[string]string{"foo": "a", "bar": "b", "baz": "c"}})
	if err != nil {
		t.Errorf("Error while rendering template: %v", err)
	}

	if strings.Count(wr.String(), "a") != 1 || strings.Count(wr.String(), "b") != 1 || strings.Count(wr.String(), "c") != 1 || strings.Count(wr.String(), ",") != 2 {
		t.Errorf("Unexpected result returned from template renderer: '%s'", wr.String())
	}
}

func TestTemplateJoinFunctionBadType(t *testing.T) {
	renderer := &TemplateRenderer{DataSources: api.EndpointMap{}}
	tmpl, err := template.New("templatebase").Funcs(renderer.TemplateFuncs()).Parse(`{{ join .bad "," }}`)
	if err != nil {
		t.Errorf("Unable to parse template for testing: %v", err)
	}
	wr := bytes.NewBufferString("")
	err = tmpl.Execute(wr, map[string]interface{}{"bad": "foo"})
	if err == nil {
		t.Errorf("Expected error rendering template, got none")
	}
}
