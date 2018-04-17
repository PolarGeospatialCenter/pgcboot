package distromux

import (
	"net/http"
	"testing"
	"text/template"
)

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
