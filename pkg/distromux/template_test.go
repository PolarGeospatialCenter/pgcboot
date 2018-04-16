package distromux

import (
	"bytes"
	"net"
	"net/http"
	"testing"
	"text/template"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/go-test/deep"
)

func TestStaticInventory(t *testing.T) {
	i := &inventory.MemoryStore{}
	SetInventoryStore(i)
	if GetInventoryStore() != i {
		t.Fatalf("Incorrect inventory store returned")
	}
}

func TestGetDataByMAC(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:8080/branch/master/foo?mac=00-de-ad-be-ef-34", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}

	store, err := inventory.NewSampleInventoryStore()
	if err != nil {
		t.Fatalf("An error ocurred while creating sample inventory store: %v", err)
	}
	SetInventoryStore(store)
	extraVars := make(map[string]interface{})
	extraVars["testkey"] = "testvalue"
	renderer := &TemplateRenderer{DefaultTemplate: "default.tmpl.yml", Network: "provisioning", DistroVars: extraVars}
	data, err := renderer.GetData(r)
	if err != nil {
		t.Fatalf("Unable to get data for request: %v", err)
	}

	d, ok := data.(*TemplateData)
	if !ok {
		t.Fatalf("The wrong data type was returned: %T", data)
	}

	if d.DistroVars == nil || d.DistroVars["testkey"].(string) != "testvalue" {
		t.Fatalf("DistroVars returned the wrong value: %v", d.DistroVars)
	}

	if d.BaseURL != "http://localhost:8080/branch/master" {
		t.Fatalf("Wrong base url generated: %s", d.BaseURL)
	}

	node := d.Node

	if node == nil {
		t.Fatalf("No node returned")
	}

	if node.InventoryID != "sample0000" {
		t.Fatalf("The wrong node was returned.")
	}
}

func TestGetDataByNodeID(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:8080/branch/master/foo?nodeid=sample0001", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}

	store, err := inventory.NewSampleInventoryStore()
	if err != nil {
		t.Fatalf("An error ocurred while creating sample inventory store: %v", err)
	}
	SetInventoryStore(store)
	extraVars := make(map[string]interface{})
	extraVars["testkey"] = "testvalue"
	renderer := &TemplateRenderer{DefaultTemplate: "default.tmpl.yml", Network: "provisioning", DistroVars: extraVars}
	data, err := renderer.GetData(r)
	if err != nil {
		t.Fatalf("Unable to get data for request: %v", err)
	}

	d, ok := data.(*TemplateData)
	if !ok {
		t.Fatalf("The wrong data type was returned: %T", data)
	}

	if d.DistroVars == nil || d.DistroVars["testkey"].(string) != "testvalue" {
		t.Fatalf("DistroVars returned the wrong value: %v", d.DistroVars)
	}

	if d.BaseURL != "http://localhost:8080/branch/master" {
		t.Fatalf("Wrong base url generated: %s", d.BaseURL)
	}

	node := d.Node

	if node == nil {
		t.Fatalf("No node returned")
	}

	if node.InventoryID != "sample0001" {
		t.Fatalf("The wrong node was returned.")
	}

	if d.RawQuery != r.URL.RawQuery {
		t.Errorf("RawQuery value not set properly")
	}
}

func TestGetDataByNodeIDRequestParams(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:8080/branch/master/foo?nodeid=sample0001&nic=eth2", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}

	store, err := inventory.NewSampleInventoryStore()
	if err != nil {
		t.Fatalf("An error ocurred while creating sample inventory store: %v", err)
	}
	SetInventoryStore(store)
	extraVars := make(map[string]interface{})
	extraVars["testkey"] = "testvalue"
	renderer := &TemplateRenderer{DefaultTemplate: "default.tmpl.yml", Network: "provisioning", DistroVars: extraVars}
	data, err := renderer.GetData(r)
	if err != nil {
		t.Fatalf("Unable to get data for request: %v", err)
	}

	d, ok := data.(*TemplateData)
	if !ok {
		t.Fatalf("The wrong data type was returned: %T", data)
	}

	if d.DistroVars == nil || d.DistroVars["testkey"].(string) != "testvalue" {
		t.Fatalf("DistroVars returned the wrong value: %v", d.DistroVars)
	}

	if d.BaseURL != "http://localhost:8080/branch/master" {
		t.Fatalf("Wrong base url generated: %s", d.BaseURL)
	}

	node := d.Node

	if node == nil {
		t.Fatalf("No node returned")
	}

	if node.InventoryID != "sample0001" {
		t.Fatalf("The wrong node was returned.")
	}

	if d.RequestParams == nil {
		t.Errorf("No request params data returned")
	}

	if v, ok := d.RequestParams["nic"]; !ok || v != "eth2" {
		t.Errorf("nic parameter not passed to template or wrong value passed in, got: %v of type %T", v, v)
	}
}

func TestTemplateSelector(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:8080/branch/master/foo?mac=00-de-ad-be-ef-34", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}

	store, err := inventory.NewSampleInventoryStore()
	if err != nil {
		t.Fatalf("An error ocurred while creating sample inventory store: %v", err)
	}
	SetInventoryStore(store)
	renderer := &TemplateRenderer{DefaultTemplate: "default.tmpl.yml", Network: "provisioning"}

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
	r, err = http.NewRequest("GET", "http://localhost:8080/branch/master/foo?nodeid=bad-node", nil)
	if err != nil {
		t.Fatalf("Unable to create request: %v", err)
	}
	// It's not our problem if the node requested doesn't exist, should return default template
	testLookup(r, []string{"default.tmpl.yml", "foo.tmpl.yml", "foo-worker.tmpl.yml"}, "default.tmpl.yml")

}

func TestTemplateFuncsIPv6HostNetStart(t *testing.T) {
	renderer := &TemplateRenderer{DefaultTemplate: "test"}
	tmpl, err := template.New("test").Funcs(renderer.TemplateFuncs()).Parse("{{ ipv6HostNet .Network .Rack .BottomU .ChassisSubIndex false .Offset }}")
	if err != nil {
		t.Errorf("unable to create template: %v", err)
	}

	var output bytes.Buffer
	data := struct {
		Network         string
		Rack            string
		BottomU         uint
		ChassisSubIndex string
		Offset          uint64
	}{
		Network:         "2001:db8::/60",
		Rack:            "ZZZZ",
		BottomU:         42,
		ChassisSubIndex: "",
		Offset:          255,
	}

	err = tmpl.ExecuteTemplate(&output, "test", data)
	if err != nil {
		t.Errorf("unable to render template: %v", err)
	}

	if output.String() != "2001:db8:0:6:683f:ea00:0:ff" {
		t.Errorf("Wrong output returned: %s", output.String())
	}

	output = bytes.Buffer{}
	data = struct {
		Network         string
		Rack            string
		BottomU         uint
		ChassisSubIndex string
		Offset          uint64
	}{
		Network:         "2001:db8::/64",
		Rack:            "xr20",
		BottomU:         31,
		ChassisSubIndex: "a",
		Offset:          20,
	}

	err = tmpl.ExecuteTemplate(&output, "test", data)
	if err != nil {
		t.Errorf("unable to render template: %v", err)
	}

	if output.String() != "2001:db8::601c:e1fa:0:14" {
		t.Errorf("Wrong output returned: %s", output.String())
	}

}

func TestTemplateFuncsIPv6HostNetEnd(t *testing.T) {
	renderer := &TemplateRenderer{DefaultTemplate: "test"}
	tmpl, err := template.New("test").Funcs(renderer.TemplateFuncs()).Parse("{{ ipv6HostNet .Network .Rack .BottomU .ChassisSubIndex true 524 }}")
	if err != nil {
		t.Errorf("unable to create template: %v", err)
	}

	var output bytes.Buffer
	data := struct {
		Network         string
		Rack            string
		BottomU         uint
		ChassisSubIndex string
	}{
		Network:         "2001:db8::/60",
		Rack:            "ZZZZ",
		BottomU:         42,
		ChassisSubIndex: "",
	}

	err = tmpl.ExecuteTemplate(&output, "test", data)
	if err != nil {
		t.Errorf("unable to render template: %v", err)
	}

	if output.String() != "2001:db8:0:6:683f:ea0f:ffff:ffff" {
		t.Errorf("Wrong output returned: %s", output.String())
	}

	output = bytes.Buffer{}
	data = struct {
		Network         string
		Rack            string
		BottomU         uint
		ChassisSubIndex string
	}{
		Network:         "2001:db8::/56",
		Rack:            "xr20",
		BottomU:         31,
		ChassisSubIndex: "a",
	}

	err = tmpl.ExecuteTemplate(&output, "test", data)
	if err != nil {
		t.Errorf("unable to render template: %v", err)
	}

	if output.String() != "2001:db8:0:60:1ce1:faff:ffff:ffff" {
		t.Errorf("Wrong output returned: %s", output.String())
	}

}

func TestTemplateFuncsIsV6(t *testing.T) {
	renderer := &TemplateRenderer{DefaultTemplate: "test"}
	tmpl, err := template.New("test").Funcs(renderer.TemplateFuncs()).Parse("{{ if isV6 .Network }}true{{ end }}")
	if err != nil {
		t.Errorf("unable to create template: %v", err)
	}

	type testCase struct {
		Network string
		V6      bool
	}

	cases := []*testCase{
		&testCase{"2001:db8::/60", true},
		&testCase{"2001:db8::", true},
		&testCase{"10.0.0.0/8", false},
		&testCase{"10.0.0.0", false},
	}

	for _, c := range cases {
		var output bytes.Buffer
		err = tmpl.ExecuteTemplate(&output, "test", c)
		if err != nil {
			t.Errorf("unable to render template for %s: %v", c.Network, err)
		}

		expected := ""
		if c.V6 {
			expected = "true"
		}
		if output.String() != expected {
			t.Errorf("Wrong output returned for %s: got '%s', expected '%s'", c.Network, output.String(), expected)
		}

	}

}

func TestIPv6HostPrefix(t *testing.T) {
	type testCase struct {
		Node           *inventorytypes.InventoryNode
		ExpectedPrefix uint64
		ExpectedErr    error
	}

	cases := []testCase{
		testCase{
			Node:           &inventorytypes.InventoryNode{Location: &inventorytypes.ChassisLocation{Rack: "xr20", BottomU: 31}, ChassisSubIndex: "a", InventoryID: "org-0001"},
			ExpectedPrefix: 0x601ce1fa,
			ExpectedErr:    nil,
		},
		testCase{
			Node:           &inventorytypes.InventoryNode{InventoryID: "org-0002"},
			ExpectedPrefix: 0xf86f0269,
			ExpectedErr:    nil,
		},
	}

	for _, c := range cases {
		prefix, err := ipv6HostPrefixBits(c.Node)
		if err != c.ExpectedErr {
			t.Errorf("Unexpected error, got: %v, expected: %v", err, c.ExpectedErr)
		}

		if prefix != c.ExpectedPrefix {
			t.Errorf("Got prefix: %x, expected: %x", prefix, c.ExpectedPrefix)
		}
	}
}

func TestGetNicConfig(t *testing.T) {
	mac, _ := net.ParseMAC("00:01:02:03:04:05")
	_, cidr1, _ := net.ParseCIDR("10.0.0.254/24")
	_, cidr2, _ := net.ParseCIDR("2001:db8::/64")
	instance := &inventorytypes.NICInstance{
		NIC: inventorytypes.NICInfo{MAC: mac, IP: net.ParseIP("10.0.0.1")},
		Network: inventorytypes.Network{
			Name: "test",
			MTU:  9000,
			Subnets: []*inventorytypes.Subnet{
				&inventorytypes.Subnet{
					Cidr:    cidr1,
					Gateway: net.ParseIP("10.0.0.254"),
					DNS:     []net.IP{net.ParseIP("10.53.53.53")},
				},
				&inventorytypes.Subnet{
					Cidr:    cidr2,
					Gateway: net.ParseIP("2001:db8::1"),
					DNS:     []net.IP{net.ParseIP("2001:db8::53::1")},
				},
			},
			Domain: "foo.bar.tld",
		},
	}
	cfg, err := GetNicConfig(instance, 0x1000000000001)
	if err != nil {
		t.Errorf("Unable to get NIC Configuration: %v", err)
	}

	compare := func(failMsg string, a, b interface{}) {
		if diff := deep.Equal(a, b); len(diff) > 0 {
			t.Error(failMsg)
			for _, d := range diff {
				t.Log(d)
			}
		}
	}
	expectedIPs := []string{"10.0.0.1/24", "2001:db8::1:0:0:1/64"}
	compare("IP list doesn't match expected value", cfg.IPs, expectedIPs)

	expectedDNS := []string{"10.53.53.53"}
	compare("DNS list doesn't match expected value", cfg.DNS, expectedDNS)

	expectedGateways := []string{"10.0.0.254", "2001:db8::1"}
	compare("Gateway list doesn't match expected value", cfg.Gateways, expectedGateways)

	expectedDomains := "foo.bar.tld"
	if cfg.Domains != expectedDomains {
		t.Errorf("Domains string doesn't match expected, got: %s, expected: %s", cfg.Domains, expectedDomains)
	}
}
