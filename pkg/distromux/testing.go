package distromux

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/PolarGeospatialCenter/pgcboot/pkg/api"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/viper"
	gock "gopkg.in/h2non/gock.v1"
)

// DistroTestConfig describes where to find tests for the distro
type DistroTestSuite struct {
	Folder string `mapstructure:"folder"`
}

type MockDataSourceCall struct {
	DataSource string           `mapstructure:"datasource"`
	Request    MockHTTPRequest  `mapstructure:"request"`
	Response   MockHTTPResponse `mapstructure:"response"`
}

func (m *MockDataSourceCall) mock(endpoints api.EndpointMap) (gock.Mock, error) {
	source, ok := endpoints[m.DataSource]
	if !ok {
		return nil, fmt.Errorf("invalid datasource specified for mock: %s", m.DataSource)
	}
	u, err := url.Parse(source.URL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse datasource url %s: %v", m.DataSource, err)
	}
	u.Path = u.Path + m.Request.Path
	u.RawQuery = m.Request.Query
	req := gock.NewRequest().SetURL(u).BodyString(m.Request.Body)
	req.Method = m.Request.Method
	res := gock.NewResponse().Status(m.Response.Status).BodyString(m.Response.Body)
	return gock.NewMock(req, res), nil
}

type MockHTTPRequest struct {
	Path    string                 `mapstructure:"path"`
	Query   string                 `mapstructure:"query"`
	Body    string                 `mapstructure:"body"`
	Method  string                 `mapstructure:"method"`
	Headers map[string]interface{} `mapstructure:"headers"`
}

func (r *MockHTTPRequest) BuildRequest(baseUrl *url.URL) (*http.Request, error) {
	uString := fmt.Sprintf("%s://%s%s%s?%s", baseUrl.Scheme, baseUrl.Hostname(), baseUrl.EscapedPath(), r.Path, r.Query)
	u, err := url.Parse(uString)
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	for field, value := range r.Headers {
		switch value.(type) {
		case string:
			headers.Add(field, value.(string))
		case []string:
			for _, v := range value.([]string) {
				headers.Add(field, v)
			}
		default:
			return nil, fmt.Errorf("Unsupported header type %T", value)
		}
	}
	req, err := http.NewRequest(r.Method, u.String(), bytes.NewBufferString(r.Body))
	if err != nil {
		return nil, err
	}
	req.Header = headers
	return req, nil
}

type MockHTTPResponse struct {
	Status int    `mapstructure:"status"`
	Body   string `mapstructure:"body"`
}

type DistroTestResult struct {
	Failed bool
	Output string
}

type DistroTestCase struct {
	InputRequest     MockHTTPRequest      `mapstructure:"request"`
	MockedData       []MockDataSourceCall `mapstructure:"mocked_data"`
	ExpectedOutput   MockHTTPResponse     `mapstructure:"expected"`
	MockedDistroVars DistroVars           `mapstructure:"vars"`
}

func LoadTestCases(testsPath string) (map[string]*DistroTestCase, error) {
	testCases := make(map[string]*DistroTestCase)
	err := filepath.Walk(testsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		cfg := viper.New()
		cfg.SetConfigFile(path)
		cfg.ReadInConfig()
		testCase := &DistroTestCase{}
		err = cfg.Unmarshal(testCase)
		testCases[path] = testCase
		return err
	})
	return testCases, err
}

func (c *DistroTestCase) Test(mux *DistroMux, endpoints api.EndpointMap) *DistroTestResult {
	// Build request
	u, _ := url.Parse("http://local")
	req, err := c.InputRequest.BuildRequest(u)
	if err != nil {
		return &DistroTestResult{Failed: true, Output: fmt.Sprintf("unable to create mock request: %v", err)}
	}
	// Mock api Endpoints
	gock.DisableNetworking()
	gock.Intercept()
	defer gock.EnableNetworking()
	defer gock.Off()

	for _, mockedCall := range c.MockedData {
		mock, err := mockedCall.mock(endpoints)
		if err != nil {
			return &DistroTestResult{Failed: true, Output: fmt.Sprintf("unable to create mock for data source call: %v", err)}
		}
		gock.Register(mock)
	}

	// mock distrovars
	req = c.MockedDistroVars.SetContextForRequest(req)

	// render response
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)

	// compare response to expected
	result := &DistroTestResult{}
	var matchingBody bool
	resultBody, err := ioutil.ReadAll(response.Result().Body)
	if err != nil {
		return &DistroTestResult{Failed: true, Output: fmt.Sprintf("unable to read result body: %v", err)}
	}
	matchingBody = (strings.TrimSpace(string(resultBody)) == strings.TrimSpace(c.ExpectedOutput.Body))
	result.Failed = (response.Result().StatusCode != c.ExpectedOutput.Status) || !matchingBody
	result.Output = fmt.Sprintf("Expected length: %d - got: %d bytes\n", len(c.ExpectedOutput.Body), len(resultBody))
	if result.Failed {
		result.Output += fmt.Sprintf("Expected status: %d - got: %d\n", c.ExpectedOutput.Status, response.Result().StatusCode)
	}
	if !matchingBody {
		dmp := diffmatchpatch.New()

		diffs := dmp.DiffMain(c.ExpectedOutput.Body, string(resultBody), false)

		result.Output += fmt.Sprintf("Body Diff:\n%s", dmp.DiffPrettyText(diffs)) +
			fmt.Sprintf("Raw Request: %v\n", req) +
			fmt.Sprintf("Raw Response: %v", response.Result())
	}
	return result
}
