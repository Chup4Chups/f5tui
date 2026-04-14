package f5

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	base     string
	user     string
	pass     string
	http     *http.Client
}

func New(host, user, pass string, insecure bool) *Client {
	return &Client{
		base: strings.TrimRight(host, "/"),
		user: user,
		pass: pass,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
			},
		},
	}
}

func (c *Client) get(path string, out any) error {
	req, err := http.NewRequest("GET", c.base+path, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.user, c.pass)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: %s", req.Method, path, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type listEnvelope[T any] struct {
	Items []T `json:"items"`
}

type VirtualServer struct {
	Name        string `json:"name"`
	Partition   string `json:"partition"`
	FullPath    string `json:"fullPath"`
	Destination string `json:"destination"`
	IPProtocol  string `json:"ipProtocol"`
	Pool        string `json:"pool"`
	Enabled     bool   `json:"-"`
	RawEnabled  string `json:"enabled,omitempty"`
	RawDisabled string `json:"disabled,omitempty"`
}

func (c *Client) VirtualServers() ([]VirtualServer, error) {
	var env listEnvelope[VirtualServer]
	if err := c.get("/mgmt/tm/ltm/virtual", &env); err != nil {
		return nil, err
	}
	for i := range env.Items {
		env.Items[i].Enabled = env.Items[i].RawDisabled == ""
	}
	return env.Items, nil
}

type Pool struct {
	Name              string `json:"name"`
	Partition         string `json:"partition"`
	FullPath          string `json:"fullPath"`
	Monitor           string `json:"monitor"`
	LoadBalancingMode string `json:"loadBalancingMode"`
	ActiveMemberCount int    `json:"activeMemberCount"`
}

func (c *Client) Pools() ([]Pool, error) {
	var env listEnvelope[Pool]
	if err := c.get("/mgmt/tm/ltm/pool", &env); err != nil {
		return nil, err
	}
	return env.Items, nil
}

type PoolMember struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	State     string `json:"state"`
	Session   string `json:"session"`
}

func (c *Client) PoolMembers(fullPath string) ([]PoolMember, error) {
	encoded := strings.ReplaceAll(fullPath, "/", "~")
	var env listEnvelope[PoolMember]
	path := fmt.Sprintf("/mgmt/tm/ltm/pool/%s/members", url.PathEscape(encoded))
	if err := c.get(path, &env); err != nil {
		return nil, err
	}
	return env.Items, nil
}

type LTMPolicy struct {
	Name      string `json:"name"`
	Partition string `json:"partition"`
	FullPath  string `json:"fullPath"`
	Status    string `json:"status"`
	Strategy  string `json:"strategy"`
}

func (c *Client) LTMPolicies() ([]LTMPolicy, error) {
	var env listEnvelope[LTMPolicy]
	if err := c.get("/mgmt/tm/ltm/policy", &env); err != nil {
		return nil, err
	}
	return env.Items, nil
}

type ASMPolicy struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Partition      string `json:"partition"`
	EnforcementMode string `json:"enforcementMode"`
	Active         bool   `json:"active"`
	VirtualServers []string `json:"virtualServers"`
}

func (c *Client) ASMPolicies() ([]ASMPolicy, error) {
	var env listEnvelope[ASMPolicy]
	if err := c.get("/mgmt/tm/asm/policies", &env); err != nil {
		return nil, err
	}
	return env.Items, nil
}
