package distromux

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/PolarGeospatialCenter/pgcboot/pkg/api"
	"github.com/go-test/deep"
	"github.com/spf13/viper"
	gock "gopkg.in/h2non/gock.v1"
)

func TestUnmarshalDistroTestCase(t *testing.T) {
	sampleTest := `---
request:
  path: /ignition
  query: id=pgc-0030
  method: GET
mocked_data:
  - datasource: node
    request:
      query: "id=pgc-0030"
      body: ""
    response:
      status: 200
      body: |
        {"InventoryID": "pgc-0030"}
expected:
  body: |
    {ignition config result}
  status: 200
`
	expectedTestCase := DistroTestCase{
		InputRequest: MockHTTPRequest{Path: "/ignition", Query: "id=pgc-0030", Method: "GET"},
		MockedData: []MockDataSourceCall{
			MockDataSourceCall{
				DataSource: "node",
				Request:    MockHTTPRequest{Query: "id=pgc-0030", Body: ""},
				Response:   MockHTTPResponse{Status: 200, Body: "{\"InventoryID\": \"pgc-0030\"}\n"},
			},
		},
		ExpectedOutput: MockHTTPResponse{Body: "{ignition config result}\n", Status: 200},
	}

	cfg := viper.New()
	cfg.SetConfigType("yaml")
	cfg.ReadConfig(bytes.NewBufferString(sampleTest))
	var result DistroTestCase
	err := cfg.Unmarshal(&result)
	if err != nil {
		t.Errorf("Unable to unmarshal test case: %v", err)
	}

	if diff := deep.Equal(result, expectedTestCase); len(diff) > 0 {
		t.Error("Unmarshaled result doesn't match expected:")
		for _, l := range diff {
			t.Error(l)
		}
	}
}

func TestMockAPICall(t *testing.T) {
	apiCall := &MockDataSourceCall{
		DataSource: "foo",
		Request:    MockHTTPRequest{Query: "id=pgc-0030", Body: ""},
		Response:   MockHTTPResponse{Status: 200, Body: "{\"InventoryID\": \"pgc-0030\"}\n"},
	}
	endpoints := api.EndpointMap{
		"foo": &api.Endpoint{URL: "http://local/v1/foo", Method: "GET"},
	}

	mock, err := apiCall.mock(endpoints)
	if err != nil {
		t.Errorf("unable to create mock: %v", err)
	}

	gock.DisableNetworking()
	defer gock.EnableNetworking()
	defer gock.Off()

	gock.Register(mock)
	httpClient := http.DefaultClient
	gock.InterceptClient(httpClient)
	gock.Intercept()

	req, _ := http.NewRequest("GET", "http://localhost/v1/foo?id=pgc-0030", bytes.NewBufferString(""))
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Errorf("unable to make request against mocked endpoint: %v", err)
	}

	resultBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("unable to read response body: %v", err)
	}
	_ = resultBody
}
