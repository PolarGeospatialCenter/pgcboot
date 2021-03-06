package distromux

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"text/template"

	"github.com/PolarGeospatialCenter/pgcboot/pkg/api"
	"github.com/go-test/deep"
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

func TestTemplateRawContentType(t *testing.T) {
	gock.DisableNetworking()
	defer gock.EnableNetworking()
	defer gock.Off() // Flush pending mocks after test execution

	e := api.EndpointMap{}
	ep := &TemplateEndpoint{TemplatePath: "foo", RawContentType: "text/yaml", ContentType: "application/json", DefaultTemplate: "default.tmpl.yml"}
	handler, err := ep.CreateHandler("../../test/data/branch/basic", "", e)
	if err != nil {
		t.Fatalf("unable to load endpoint for testing: %v", err)
	}

	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "https://test.local/branch/dev/foo?raw", nil)
	if err != nil {
		t.Errorf("unable to create test request: %v", err)
	}

	ctx := NewDistroVarsContext(request.Context(), DistroVars{})
	handler.ServeHTTP(response, request.WithContext(ctx))

	if response.Result().StatusCode != http.StatusOK {
		t.Errorf("Got non-OK status: %d", response.Result().StatusCode)
		body, _ := ioutil.ReadAll(response.Result().Body)
		t.Errorf("Result Body: %s", body)
	}

	contentType := response.Header().Get("Content-type")
	if contentType != "text/yaml" {
		t.Errorf("Got wrong raw content-type: %s", contentType)
	}
}

func TestTemplatePostRenderContentType(t *testing.T) {
	gock.DisableNetworking()
	defer gock.EnableNetworking()
	defer gock.Off() // Flush pending mocks after test execution

	e := api.EndpointMap{}
	ep := &TemplateEndpoint{TemplatePath: "foo", RawContentType: "text/yaml", ContentType: "application/json", DefaultTemplate: "default.tmpl.yml", PostRender: []string{"cat"}}
	handler, err := ep.CreateHandler("../../test/data/branch/basic", "", e)
	if err != nil {
		t.Fatalf("unable to load endpoint for testing: %v", err)
	}

	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "https://test.local/branch/dev/foo", nil)
	if err != nil {
		t.Errorf("unable to create test request: %v", err)
	}

	ctx := NewDistroVarsContext(request.Context(), DistroVars{})
	handler.ServeHTTP(response, request.WithContext(ctx))

	if response.Result().StatusCode != http.StatusOK {
		t.Errorf("Got non-OK status: %d", response.Result().StatusCode)
		body, _ := ioutil.ReadAll(response.Result().Body)
		t.Errorf("Result Body: %s", body)
	}

	contentType := response.Header().Get("Content-type")
	if contentType != "application/json" {
		t.Errorf("Got wrong raw content-type: %s", contentType)
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

func TestTemplateNetworkCidrContains(t *testing.T) {
	cases := []struct {
		name           string
		cidr           string
		ips            interface{}
		expectedResult []string
		expectedErr    error
	}{
		{
			name:           "Success 10/8",
			cidr:           "10.0.0.0/8",
			ips:            []string{"10.0.0.1", "192.168.0.1", "2001:db8::1"},
			expectedResult: []string{"10.0.0.1"},
			expectedErr:    nil,
		},
		{
			name:           "Success 2001:db8::/64",
			cidr:           "2001:db8::/64",
			ips:            []string{"10.0.0.1", "192.168.0.1", "2001:db8::1", "2001:db8:0:4::1"},
			expectedResult: []string{"2001:db8::1"},
			expectedErr:    nil,
		},
		{
			name:           "Fail, bad cidr",
			cidr:           "",
			ips:            []string{"10.0.0.1", "192.168.0.1", "2001:db8::1", "2001:db8:0:4::1"},
			expectedResult: []string{},
			expectedErr:    &net.ParseError{Type: "CIDR address"},
		},
		{
			name:           "Success, badly formatted IP",
			cidr:           "2001:db8::/64",
			ips:            []string{"10.0.0.1", "192.168.0.1", "asdf", "", "2001:db8::1", "2001:db8:0:4::1"},
			expectedResult: []string{"2001:db8::1"},
			expectedErr:    nil,
		},
		{
			name:           "Success, cidr IP",
			cidr:           "2001:db8::/64",
			ips:            []string{"10.0.0.1", "192.168.0.1", "asdf", "", "2001:db8::1/64", "2001:db8:0:4::1"},
			expectedResult: []string{"2001:db8::1"},
			expectedErr:    nil,
		},
		{
			name:           "Success, interface array",
			cidr:           "2001:db8::/64",
			ips:            []interface{}{"10.0.0.1", "192.168.0.1", "asdf", 6, "2001:db8::1", "2001:db8:0:4::1"},
			expectedResult: []string{"2001:db8::1"},
			expectedErr:    nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(st *testing.T) {
			r, err := TemplateNetworkCidrContains(c.cidr, c.ips)
			if diff := deep.Equal(r, c.expectedResult); len(diff) > 0 {
				st.Error("Expected result mismatch:")
				for _, l := range diff {
					st.Error(l)
				}
			}

			if err != c.expectedErr && err.Error() != c.expectedErr.Error() {
				st.Errorf("Expected error '%v' got '%s'", c.expectedErr, err)
			}
		})
	}
}
func TestTemplateCidrContains(t *testing.T) {
	renderer := &TemplateRenderer{DataSources: api.EndpointMap{}}
	tmpl, err := template.New("templatebase").Funcs(renderer.TemplateFuncs()).Parse(`{{ $ips := applyCidrMask "2001:db8::/64" .ips }}{{ index $ips 0 }}`)
	if err != nil {
		t.Errorf("Unable to parse template for testing: %v", err)
	}
	wr := bytes.NewBufferString("")
	err = tmpl.Execute(wr, map[string]interface{}{"ips": []string{"10.0.0.1", "192.168.0.1", "asdf", "", "2001:db8:0:0::1", "2001:db8:0:4::1"}})
	if err != nil {
		t.Errorf("Error while rendering template: %v", err)
	}

	if wr.String() != "2001:db8::1" {
		t.Errorf("Unexpected result returned from template renderer: '%s'", wr.String())
	}

}

func TestSprigDict(t *testing.T) {
	renderer := &TemplateRenderer{DataSources: api.EndpointMap{}}
	tmpl, err := template.New("templatebase").Funcs(renderer.TemplateFuncs()).Parse(`{{ $d := dict "foo" "fooval" "bar" "barval" }}{{ $d.foo }}`)
	if err != nil {
		t.Errorf("Unable to parse template for testing: %v", err)
	}
	wr := bytes.NewBufferString("")
	err = tmpl.Execute(wr, "")
	if err != nil {
		t.Errorf("Error while rendering template: %v", err)
	}

	if wr.String() != "fooval" {
		t.Errorf("Unexpected result returned from template renderer: '%s'", wr.String())
	}

}
func TestGetTemplateBaseURLXForwardedProto(t *testing.T) {
	renderer := &TemplateRenderer{DataSources: api.EndpointMap{}}
	testUrl, _ := url.Parse("http://test.local/foo/bar")
	headers := http.Header{}
	headers.Add("X-Forwarded-Proto", "https")
	baseUrl, err := renderer.getBaseURL(&http.Request{
		Method: http.MethodGet,
		URL:    testUrl,
		Host:   "test.local",
		Header: headers,
	})
	if err != nil {
		t.Errorf("Error getting base url: %v", baseUrl)
	}
	if baseUrl != "https://test.local/foo" {
		t.Errorf("Wrong base url returned: %s", baseUrl)
	}
}

func TestGetTemplateBaseURL(t *testing.T) {
	renderer := &TemplateRenderer{DataSources: api.EndpointMap{}}
	testUrl, _ := url.Parse("http://test.local/foo/bar")
	baseUrl, err := renderer.getBaseURL(&http.Request{
		Method: http.MethodGet,
		URL:    testUrl,
		Host:   "test.local",
	})
	if err != nil {
		t.Errorf("Error getting base url: %v", baseUrl)
	}
	if baseUrl != "http://test.local/foo" {
		t.Errorf("Wrong base url returned: %s", baseUrl)
	}
}
