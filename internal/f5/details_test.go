package f5

import (
	"net/http"
	"strings"
	"testing"
)

func TestVirtualServerDetailParsing(t *testing.T) {
	var gotPath string
	body := `{
		"name":"vs_web_http",
		"partition":"Common",
		"fullPath":"/Common/vs_web_http",
		"destination":"/Common/10.0.0.10:80",
		"ipProtocol":"tcp",
		"pool":"/Common/pool_web",
		"enabled":"true",
		"rules":["/Common/_sys_https_redirect"],
		"profilesReference":{"items":[
			{"name":"http","partition":"Common","fullPath":"/Common/http","context":"all"}
		]},
		"policiesReference":{"items":[
			{"name":"policy_host_routing","fullPath":"/Common/policy_host_routing"}
		]}
	}`
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(body))
	})
	defer srv.Close()

	vs, err := New(srv.URL, "u", "p", false).VirtualServerDetail("/Common/vs_web_http")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "~Common~vs_web_http") {
		t.Errorf("expected tilde-encoded path, got %q", gotPath)
	}
	if !vs.Enabled {
		t.Errorf("vs should be enabled")
	}
	if len(vs.ProfilesReference.Items) != 1 || vs.ProfilesReference.Items[0].Name != "http" {
		t.Errorf("profiles not parsed: %+v", vs.ProfilesReference.Items)
	}
	if len(vs.PoliciesReference.Items) != 1 || vs.PoliciesReference.Items[0].FullPath != "/Common/policy_host_routing" {
		t.Errorf("policies not parsed: %+v", vs.PoliciesReference.Items)
	}
	if len(vs.Rules) != 1 {
		t.Errorf("rules not parsed")
	}
}

func TestPoolDetailParsing(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"name":"pool_web","partition":"Common","fullPath":"/Common/pool_web",
			"monitor":"/Common/http","loadBalancingMode":"round-robin","activeMemberCount":2,
			"members":[
				{"name":"10.0.0.1:80","address":"10.0.0.1","state":"up","session":"monitor-enabled"},
				{"name":"10.0.0.2:80","address":"10.0.0.2","state":"down","session":"user-disabled"}
			]
		}`))
	})
	defer srv.Close()
	p, err := New(srv.URL, "u", "p", false).PoolDetail("/Common/pool_web")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Members) != 2 || p.Members[1].State != "down" {
		t.Errorf("members not parsed: %+v", p.Members)
	}
}

func TestLTMPolicyDetailParsing(t *testing.T) {
	body := `{
		"name":"policy_host_routing","partition":"Common","fullPath":"/Common/policy_host_routing",
		"status":"published","strategy":"/Common/first-match",
		"rulesReference":{"items":[
			{"name":"rule_api","ordinal":0,
			 "conditionsReference":{"items":[
				{"httpHost":true,"equals":true,"request":true,"values":["api.example.com"]}
			 ]},
			 "actionsReference":{"items":[
				{"forward":true,"request":true,"pool":"/Common/pool_api"}
			 ]}}
		]}
	}`
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	defer srv.Close()
	p, err := New(srv.URL, "u", "p", false).LTMPolicyDetail("/Common/policy_host_routing")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.RulesReference.Items) != 1 {
		t.Fatalf("want 1 rule, got %d", len(p.RulesReference.Items))
	}
	rule := p.RulesReference.Items[0]
	if len(rule.ConditionsReference.Items) != 1 || !rule.ConditionsReference.Items[0].HTTPHost {
		t.Errorf("condition not parsed: %+v", rule.ConditionsReference.Items)
	}
	if len(rule.ActionsReference.Items) != 1 || rule.ActionsReference.Items[0].Pool != "/Common/pool_api" {
		t.Errorf("action not parsed: %+v", rule.ActionsReference.Items)
	}
}

func TestConditionDescribe(t *testing.T) {
	c := PolicyCondition{HTTPHost: true, Equals: true, Values: []string{"api.example.com"}}
	if got := c.Describe(); got != "http-host equals [api.example.com]" {
		t.Errorf("got %q", got)
	}
	c2 := PolicyCondition{GeoIP: true, Not: true, Equals: true, Values: []string{"FR", "DE"}}
	if got := c2.Describe(); got != "NOT geoip equals [FR, DE]" {
		t.Errorf("got %q", got)
	}
}

func TestActionDescribe(t *testing.T) {
	cases := []struct {
		a    PolicyAction
		want string
	}{
		{PolicyAction{Forward: true, Pool: "/Common/pool_api"}, "forward to pool /Common/pool_api"},
		{PolicyAction{Redirect: true, Location: "https://example.com"}, "redirect to https://example.com"},
		{PolicyAction{Reset: true}, "reset connection"},
		{PolicyAction{Replace: true, HTTPUri: true, Value: "/v2/"}, "rewrite http-uri to /v2/"},
	}
	for _, c := range cases {
		if got := c.a.Describe(); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestASMPolicyDetailAndSubcollections(t *testing.T) {
	var detailPath, urlsPath, paramsPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/urls"):
			urlsPath = r.URL.Path
			_, _ = w.Write([]byte(`{"items":[{"name":"/login","protocol":"https","method":"POST","type":"explicit","performStaging":false}]}`))
		case strings.HasSuffix(r.URL.Path, "/parameters"):
			paramsPath = r.URL.Path
			_, _ = w.Write([]byte(`{"items":[{"name":"username","type":"explicit","level":"global","valueType":"user-input","performStaging":false}]}`))
		default:
			detailPath = r.URL.Path
			_, _ = w.Write([]byte(`{"id":"abc123","name":"asm_web_policy","partition":"Common","enforcementMode":"blocking","active":true,"virtualServers":["/Common/vs_web_https"],"learningMode":"manual","signatureSets":[{"name":"Generic","alarm":true,"block":true}]}`))
		}
	})
	defer srv.Close()

	c := New(srv.URL, "u", "p", false)
	p, err := c.ASMPolicyDetail("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "asm_web_policy" || !p.Active || p.EnforcementMode != "blocking" {
		t.Errorf("detail parse wrong: %+v", p)
	}
	if detailPath != "/mgmt/tm/asm/policies/abc123" {
		t.Errorf("detail path = %q", detailPath)
	}

	urls, err := c.ASMPolicyURLs("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 || urls[0].Name != "/login" {
		t.Errorf("urls parse wrong: %+v", urls)
	}
	if urlsPath != "/mgmt/tm/asm/policies/abc123/urls" {
		t.Errorf("urls path = %q", urlsPath)
	}

	params, err := c.ASMPolicyParameters("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if len(params) != 1 || params[0].Name != "username" {
		t.Errorf("params parse wrong: %+v", params)
	}
	if paramsPath != "/mgmt/tm/asm/policies/abc123/parameters" {
		t.Errorf("params path = %q", paramsPath)
	}
}
