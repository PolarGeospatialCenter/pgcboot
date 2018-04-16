package distromux

import (
	"fmt"
	"hash/crc32"
	"log"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/PolarGeospatialCenter/inventory/pkg/inventory"
	inventorytypes "github.com/PolarGeospatialCenter/inventory/pkg/inventory/types"
	"github.com/PolarGeospatialCenter/ipxeserver/pkg/handler/pipe"
	"github.com/PolarGeospatialCenter/ipxeserver/pkg/handler/template"
	"github.com/azenk/iputils"
)

var distromuxInventory inventory.InventoryStore

// SetInventoryStore sets the InventoryStore to be used globally
func SetInventoryStore(d inventory.InventoryStore) {
	distromuxInventory = d
}

// GetInventoryStore gets the global InventoryStore
func GetInventoryStore() inventory.InventoryStore {
	return distromuxInventory
}

// TemplateData is the struct that will be passed into the template at render time
type TemplateData struct {
	Node          *inventorytypes.InventoryNode
	BaseURL       string
	DistroVars    map[string]interface{}
	RequestParams map[string]interface{}
	RawQuery      string
}

// TemplateRenderer implements the RenderManager interface.
type TemplateRenderer struct {
	DefaultTemplate string
	Network         string
	DistroVars      map[string]interface{}
}

func (tr *TemplateRenderer) getBaseURL(r *http.Request) (string, error) {
	relpath := ""
	if r.URL.Path[0] == '/' {
		pathels := strings.Split(r.URL.Path[1:], "/")
		relpath = strings.Join(pathels[0:2], "/")
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	host := r.Host

	u, err := url.Parse(fmt.Sprintf("%s://%s/%s", scheme, host, relpath))
	return u.String(), err
}

func (tr *TemplateRenderer) lookupNodeByMAC(mac net.HardwareAddr) (*inventorytypes.InventoryNode, error) {
	nodes, err := GetInventoryStore().Nodes()
	if err != nil {
		return nil, err
	}

	if tr.Network == "" {
		tr.Network = "provisioning"
	}

	for _, node := range nodes {
		if net, ok := node.Networks[tr.Network]; ok && net.NIC.MAC.String() == mac.String() {
			return node, nil
		}
	}
	return nil, templatehandler.ErrNotFound{Message: "No node exists with that MAC address"}
}

func (tr *TemplateRenderer) lookupNodeByID(id string) (*inventorytypes.InventoryNode, error) {
	nodes, err := GetInventoryStore().Nodes()
	if err != nil {
		return nil, err
	}

	node, ok := nodes[id]
	if !ok {
		return nil, templatehandler.ErrNotFound{Message: "No node exists with that id"}
	}

	return node, nil
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

	var node *inventorytypes.InventoryNode

	nodeid, ok := query["nodeid"]
	if ok {
		log.Printf("Looking up node by nodeid: %s", nodeid[0])
		node, err = tr.lookupNodeByID(nodeid[0])
		if err != nil {
			return nil, err
		}
		log.Printf("Found node: %s", node.ID())
		templateData.Node = node
		return templateData, nil
	}

	mac, ok := query["mac"]
	if ok {
		log.Printf("Looking up node by mac address: %s", mac[0])
		macAddr, err := net.ParseMAC(mac[0])
		if err != nil {
			return nil, err
		}
		node, err = tr.lookupNodeByMAC(macAddr)
		if err != nil {
			return nil, err
		}
		log.Printf("Found node: %s", node.ID())
		templateData.Node = node
		return templateData, nil
	}

	log.Printf("Failed to find node, returning data without node attached")
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

	// No node specified in request, return default template
	if data.Node == nil {
		return tr.DefaultTemplate, nil
	}
	node := data.Node

	templateMap := templateNames(t)

	if node.Role != "" {
		templateName, ok := templateMap[node.Role]
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

func ipv6HostNet(networkCidr string, rack string, bottomU uint, chassisSubIdx string, onesFill bool, hostBits uint64) (string, error) {
	ip, network, err := net.ParseCIDR(networkCidr)
	if err != nil {
		return "", err
	}

	rackInt, err := strconv.ParseUint(rack[len(rack)-4:], 36, 32)
	if err != nil {
		return "", err
	}

	subChassisInt := uint64(0)
	if chassisSubIdx != "" {
		subChassisInt, err = strconv.ParseUint(chassisSubIdx, 16, 32)
		if err != nil {
			return "", err
		}
	}

	locationBits := rackInt << 10
	locationBits |= (uint64(bottomU) << 4) & 0x03f0
	locationBits |= subChassisInt & 0x0f

	startoffset, _ := network.Mask.Size()

	newIp, err := iputils.SetBits(ip, locationBits, uint(startoffset), 32)
	if err != nil {
		return "", err
	}
	newIp, err = iputils.SetBits(newIp, hostBits, uint(startoffset+32), uint(128-(startoffset+32)))

	if err != nil {
		return "", err
	}

	if onesFill {
		newIp, err = iputils.SetBits(newIp, ^uint64(0), uint(startoffset+32), uint(128-(startoffset+32)))
	}
	return newIp.String(), err
}

// returns true if ip is either an ipv6 ip
func isV6(ipString string) (bool, error) {
	ip := net.ParseIP(ipString)
	if ip != nil {
		return ip.To4() == nil, nil
	}

	ip, _, err := net.ParseCIDR(ipString)
	if err != nil {
		return false, fmt.Errorf("unable to parse ip %s: %v", ipString, err)
	}
	return ip.To4() == nil, nil
}

func ipv6HostPrefixBits(node *inventorytypes.InventoryNode) (uint64, error) {
	if node.Location != nil {
		rack := node.Location.Rack
		bottomU := node.Location.BottomU
		chassisSubIdx := node.ChassisSubIndex

		rackInt, err := strconv.ParseUint(rack[len(rack)-4:], 36, 32)
		if err != nil {
			return 0, err
		}

		subChassisInt := uint64(0)
		if chassisSubIdx != "" {
			subChassisInt, err = strconv.ParseUint(chassisSubIdx, 16, 32)
			if err != nil {
				return 0, err
			}
		}

		locationBits := rackInt << 10
		locationBits |= (uint64(bottomU) << 4) & 0x03f0
		locationBits |= subChassisInt & 0x0f
		return locationBits, nil
	}

	// if location is unset, return crc32 of inventoryID with MSB forced to 1
	idCrc32 := crc32.NewIEEE()
	idCrc32.Write([]byte(node.InventoryID))
	return uint64(idCrc32.Sum32() | 0x80000000), nil
}

type NICConfig struct {
	IPs      []string
	DNS      []string
	Gateways []string
	Domains  string
}

func GetNicConfig(instance *inventorytypes.NICInstance, ipv6HostBits uint64) (*NICConfig, error) {
	cfg := &NICConfig{}
	for _, subnet := range instance.Network.Subnets {
		var newIP net.IPNet
		newIP.Mask = make([]byte, len(subnet.Cidr.Mask))
		copy(newIP.Mask, subnet.Cidr.Mask)
		if subnet.Cidr.Contains(instance.NIC.IP) {
			newIP.IP = instance.NIC.IP
		} else {
			if subnet.Cidr.IP.To4() == nil {
				networkBits, _ := subnet.Cidr.Mask.Size()
				var err error
				newIP.IP, err = iputils.SetBits(subnet.Cidr.IP, ipv6HostBits, uint(networkBits), uint(128-networkBits))
				if err != nil {
					return nil, err
				}
			}
		}
		log.Print(newIP.Mask)
		cfg.IPs = append(cfg.IPs, newIP.String())

		for _, dns := range subnet.DNS {
			if dns != nil {
				cfg.DNS = append(cfg.DNS, dns.String())
			}
			log.Print(cfg.DNS)
		}

		if subnet.Gateway != nil {
			cfg.Gateways = append(cfg.Gateways, subnet.Gateway.String())
			log.Print(cfg.Gateways)
		}
	}
	cfg.Domains = instance.Network.Domain
	return cfg, nil
}

func (tr *TemplateRenderer) TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"ipv6HostNet": ipv6HostNet,
		"isV6":        isV6,
	}
}

// TemplateEndpoint describes the configuration of an endpoint based on golang
// templates.
type TemplateEndpoint struct {
	TemplatePath    string   `mapstructure:"template_path"`
	ContentType     string   `mapstructure:"content_type"`
	DefaultTemplate string   `mapstructure:"default_template"`
	Network         string   `mapstructure:"network"`
	PostRender      []string `mapstructure:"post_render"`
}

// CreateHandler returns a handler for the endpoint described by this configuration
func (e *TemplateEndpoint) CreateHandler(basepath string, _ string, distroVars map[string]interface{}) (http.Handler, error) {
	var h http.Handler
	headers := make(map[string]string)
	headers["Content-type"] = e.ContentType
	tr := &TemplateRenderer{DefaultTemplate: e.DefaultTemplate, Network: e.Network, DistroVars: distroVars}
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
