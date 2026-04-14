package f5

import (
	"fmt"
	"strings"
)

// encodeLTMPath turns "/Common/pool_web" into "~Common~pool_web" for iControl REST URLs.
func encodeLTMPath(fullPath string) string {
	return strings.ReplaceAll(fullPath, "/", "~")
}

// ---- Virtual Server detail ----

type ProfileRef struct {
	Name      string `json:"name"`
	Partition string `json:"partition"`
	FullPath  string `json:"fullPath"`
	Context   string `json:"context"`
}

type PolicyRef struct {
	Name     string `json:"name"`
	FullPath string `json:"fullPath"`
}

type VirtualServerDetail struct {
	Name        string   `json:"name"`
	Partition   string   `json:"partition"`
	FullPath    string   `json:"fullPath"`
	Destination string   `json:"destination"`
	IPProtocol  string   `json:"ipProtocol"`
	Pool        string   `json:"pool"`
	Source      string   `json:"source"`
	SNAT        string   `json:"snat"`
	Description string   `json:"description"`
	Rules       []string `json:"rules"`
	RawEnabled  string   `json:"enabled,omitempty"`
	RawDisabled string   `json:"disabled,omitempty"`
	Enabled     bool     `json:"-"`

	ProfilesReference struct {
		Items []ProfileRef `json:"items"`
	} `json:"profilesReference"`
	PoliciesReference struct {
		Items []PolicyRef `json:"items"`
	} `json:"policiesReference"`
}

func (c *Client) VirtualServerDetail(fullPath string) (*VirtualServerDetail, error) {
	var out VirtualServerDetail
	if err := c.get("/mgmt/tm/ltm/virtual/"+encodeLTMPath(fullPath), &out); err != nil {
		return nil, err
	}
	out.Enabled = out.RawDisabled == ""
	return &out, nil
}

// ---- Pool detail ----

type PoolDetail struct {
	Name              string       `json:"name"`
	Partition         string       `json:"partition"`
	FullPath          string       `json:"fullPath"`
	Monitor           string       `json:"monitor"`
	LoadBalancingMode string       `json:"loadBalancingMode"`
	ActiveMemberCount int          `json:"activeMemberCount"`
	Description       string       `json:"description"`
	Members           []PoolMember `json:"members"`
}

func (c *Client) PoolDetail(fullPath string) (*PoolDetail, error) {
	var out PoolDetail
	if err := c.get("/mgmt/tm/ltm/pool/"+encodeLTMPath(fullPath), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---- LTM policy detail with rules/conditions/actions ----

type PolicyCondition struct {
	Name       string   `json:"name"`
	Request    bool     `json:"request"`
	Response   bool     `json:"response"`
	Not        bool     `json:"not"`
	HTTPHost   bool     `json:"httpHost"`
	HTTPUri    bool     `json:"httpUri"`
	HTTPMethod bool     `json:"httpMethod"`
	HTTPHeader bool     `json:"httpHeader"`
	GeoIP      bool     `json:"geoip"`
	Equals     bool     `json:"equals"`
	StartsWith bool     `json:"startsWith"`
	EndsWith   bool     `json:"endsWith"`
	Contains   bool     `json:"contains"`
	Values     []string `json:"values"`
}

// Describe returns a human-readable summary of the condition (e.g. "http-host equals api.example.com").
func (c PolicyCondition) Describe() string {
	field := pickLabel([]labeledBool{
		{"http-host", c.HTTPHost},
		{"http-uri", c.HTTPUri},
		{"http-method", c.HTTPMethod},
		{"http-header", c.HTTPHeader},
		{"geoip", c.GeoIP},
	})
	op := pickLabel([]labeledBool{
		{"equals", c.Equals},
		{"starts-with", c.StartsWith},
		{"ends-with", c.EndsWith},
		{"contains", c.Contains},
	})
	neg := ""
	if c.Not {
		neg = "NOT "
	}
	return fmt.Sprintf("%s%s %s [%s]", neg, field, op, strings.Join(c.Values, ", "))
}

type PolicyAction struct {
	Name     string `json:"name"`
	Request  bool   `json:"request"`
	Response bool   `json:"response"`
	Forward  bool   `json:"forward"`
	Redirect bool   `json:"redirect"`
	Reset    bool   `json:"reset"`
	Replace  bool   `json:"replace"`
	HTTPUri  bool   `json:"httpUri"`
	Pool     string `json:"pool"`
	Location string `json:"location"`
	Value    string `json:"value"`
}

func (a PolicyAction) Describe() string {
	switch {
	case a.Forward && a.Pool != "":
		return "forward to pool " + a.Pool
	case a.Redirect && a.Location != "":
		return "redirect to " + a.Location
	case a.Reset:
		return "reset connection"
	case a.Replace && a.HTTPUri && a.Value != "":
		return "rewrite http-uri to " + a.Value
	}
	return a.Name
}

type PolicyRule struct {
	Name                string `json:"name"`
	Ordinal             int    `json:"ordinal"`
	Description         string `json:"description"`
	ConditionsReference struct {
		Items []PolicyCondition `json:"items"`
	} `json:"conditionsReference"`
	ActionsReference struct {
		Items []PolicyAction `json:"items"`
	} `json:"actionsReference"`
}

type LTMPolicyDetail struct {
	Name           string `json:"name"`
	Partition      string `json:"partition"`
	FullPath       string `json:"fullPath"`
	Status         string `json:"status"`
	Strategy       string `json:"strategy"`
	Description    string `json:"description"`
	RulesReference struct {
		Items []PolicyRule `json:"items"`
	} `json:"rulesReference"`
}

func (c *Client) LTMPolicyDetail(fullPath string) (*LTMPolicyDetail, error) {
	var out LTMPolicyDetail
	if err := c.get("/mgmt/tm/ltm/policy/"+encodeLTMPath(fullPath), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---- ASM policy detail / urls / parameters ----

type ASMSignatureSet struct {
	Name  string `json:"name"`
	Alarm bool   `json:"alarm"`
	Block bool   `json:"block"`
}

type ASMPolicyDetail struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Partition           string            `json:"partition"`
	EnforcementMode     string            `json:"enforcementMode"`
	Active              bool              `json:"active"`
	VirtualServers      []string          `json:"virtualServers"`
	Description         string            `json:"description"`
	ApplicationLanguage string            `json:"applicationLanguage"`
	CaseInsensitive     bool              `json:"caseInsensitive"`
	EnablePassiveMode   bool              `json:"enablePassiveMode"`
	ProtocolIndependent bool              `json:"protocolIndependent"`
	SignatureStaging    bool              `json:"signatureStaging"`
	LearningMode        string            `json:"learningMode"`
	SignatureSets       []ASMSignatureSet `json:"signatureSets"`
}

func (c *Client) ASMPolicyDetail(id string) (*ASMPolicyDetail, error) {
	var out ASMPolicyDetail
	if err := c.get("/mgmt/tm/asm/policies/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type ASMURL struct {
	Name           string `json:"name"`
	Protocol       string `json:"protocol"`
	Method         string `json:"method"`
	Type           string `json:"type"`
	PerformStaging bool   `json:"performStaging"`
}

func (c *Client) ASMPolicyURLs(id string) ([]ASMURL, error) {
	var env listEnvelope[ASMURL]
	if err := c.get("/mgmt/tm/asm/policies/"+id+"/urls", &env); err != nil {
		return nil, err
	}
	return env.Items, nil
}

type ASMParameter struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	Level          string `json:"level"`
	ValueType      string `json:"valueType"`
	PerformStaging bool   `json:"performStaging"`
}

func (c *Client) ASMPolicyParameters(id string) ([]ASMParameter, error) {
	var env listEnvelope[ASMParameter]
	if err := c.get("/mgmt/tm/asm/policies/"+id+"/parameters", &env); err != nil {
		return nil, err
	}
	return env.Items, nil
}

// ---- helpers ----

type labeledBool struct {
	label string
	v     bool
}

func pickLabel(items []labeledBool) string {
	for _, it := range items {
		if it.v {
			return it.label
		}
	}
	return "?"
}
