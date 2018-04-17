package distromux

import (
	"net/http"
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

	testVars := make(map[string]interface{})
	testVars["kube_version"] = "1.9.0"

	renderer := &TemplateRenderer{DistroVars: testVars}

	rawData, err := renderer.GetData(r)
	data, ok := rawData.(*TemplateData)
	if !ok {
		t.Errorf("got unexpected data type from renderer: %T", data)
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

func TestTemplateSelector(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:8080/branch/master/foo?role=worker", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}

	renderer := &TemplateRenderer{DefaultTemplate: "default.tmpl.yml", FileNameTemplate: `{{ index .RequestParams "role" }}`}

	testLookup := func(req *http.Request, templates []string, expectedTemplate string) {
		tmpl := &template.Template{}
		var err error
		for _, templateName := range templates {
			tmpl, err = tmpl.New(templateName).Parse("Test")
			if err != nil {
				t.Errorf("Unable to create test template %s: %v", templateName, err)
			}
		}
		name, err := renderer.TemplateSelector(req, tmpl)
		if err != nil {
			t.Errorf("Unable to get template for request: %v", err)
		}

		if name != expectedTemplate {
			t.Errorf("The wrong template was returned: %s, expecting: %s", name, expectedTemplate)
		}
	}
	testLookup(r, []string{"default.tmpl.yml", "master.tmpl.yml", "bar-role.tmpl.yml"}, "default.tmpl.yml")
	testLookup(r, []string{"default.tmpl.yml", "master.tmpl.yml", "bar-role.tmpl.yml", "worker.tmpl.yml"}, "worker.tmpl.yml")

	// Test lookup for no node
	r, err = http.NewRequest("GET", "http://localhost:8080/branch/master/foo", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	testLookup(r, []string{"default.tmpl.yml", "foo.tmpl.yml", "foo-worker.tmpl.yml"}, "default.tmpl.yml")

	// Test lookup for bad node
	r, err = http.NewRequest("GET", "http://localhost:8080/branch/master/foo?role=bad-role", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	// It's not our problem if the node requested doesn't exist, should return default template
	testLookup(r, []string{"default.tmpl.yml", "foo.tmpl.yml", "foo-worker.tmpl.yml"}, "default.tmpl.yml")

}

func TestTemplateAPICall(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	e := api.EndpointMap{"test": &api.Endpoint{URL: "https://api.local/v1/foo", Method: http.MethodGet}}
	gock.New("https://api.local/v1").
		Get("/foo").
		Reply(200).
		JSON(map[string]string{"foo": "bar"})

	renderer := &TemplateRenderer{DataSources: e}
	_ = renderer

}
