package f5

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestBasicAuthHeaderSent(t *testing.T) {
	var gotUser, gotPass string
	var ok bool
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, ok = r.BasicAuth()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	})
	defer srv.Close()

	c := New(srv.URL, "admin", "s3cret", false)
	if _, err := c.VirtualServers(); err != nil {
		t.Fatalf("VirtualServers: %v", err)
	}
	if !ok || gotUser != "admin" || gotPass != "s3cret" {
		t.Fatalf("basic auth not sent correctly: ok=%v user=%q pass=%q", ok, gotUser, gotPass)
	}
}

func TestVirtualServersParsing(t *testing.T) {
	body := `{"items":[
		{"name":"vs_a","partition":"Common","fullPath":"/Common/vs_a","destination":"/Common/1.1.1.1:80","ipProtocol":"tcp","pool":"/Common/p","enabled":"true"},
		{"name":"vs_b","partition":"Common","fullPath":"/Common/vs_b","destination":"/Common/1.1.1.2:80","ipProtocol":"tcp","pool":"/Common/p","disabled":"true"}
	]}`
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mgmt/tm/ltm/virtual" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(body))
	})
	defer srv.Close()

	items, err := New(srv.URL, "u", "p", false).VirtualServers()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	if !items[0].Enabled {
		t.Errorf("vs_a should be enabled")
	}
	if items[1].Enabled {
		t.Errorf("vs_b should be disabled (RawDisabled=%q)", items[1].RawDisabled)
	}
}

func TestPoolsParsing(t *testing.T) {
	body := `{"items":[{"name":"p1","partition":"Common","fullPath":"/Common/p1","monitor":"/Common/http","loadBalancingMode":"round-robin","activeMemberCount":5}]}`
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	defer srv.Close()
	items, err := New(srv.URL, "u", "p", false).Pools()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ActiveMemberCount != 5 || items[0].LoadBalancingMode != "round-robin" {
		t.Fatalf("unexpected parse: %+v", items)
	}
}

func TestPoolMembersEncodesFullPath(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"items":[{"name":"10.0.0.1:80","address":"10.0.0.1","state":"up","session":"monitor-enabled"}]}`))
	})
	defer srv.Close()

	items, err := New(srv.URL, "u", "p", false).PoolMembers("/Common/pool_web")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1 member, got %d", len(items))
	}
	if !strings.Contains(gotPath, "~Common~pool_web") {
		t.Errorf("expected tilde-encoded path, got %q", gotPath)
	}
}

func TestLTMPoliciesParsing(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"name":"p","partition":"Common","fullPath":"/Common/p","status":"published","strategy":"/Common/first-match"}]}`))
	})
	defer srv.Close()
	items, err := New(srv.URL, "u", "p", false).LTMPolicies()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != "published" {
		t.Fatalf("unexpected: %+v", items)
	}
}

func TestASMPoliciesParsing(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":"x","name":"asm1","partition":"Common","enforcementMode":"blocking","active":true,"virtualServers":["/Common/vs"]}]}`))
	})
	defer srv.Close()
	items, err := New(srv.URL, "u", "p", false).ASMPolicies()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !items[0].Active || items[0].EnforcementMode != "blocking" {
		t.Fatalf("unexpected: %+v", items)
	}
}

func TestErrorResponse(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":401,"message":"unauthorized"}`, http.StatusUnauthorized)
	})
	defer srv.Close()
	_, err := New(srv.URL, "u", "p", false).VirtualServers()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("error should contain server message, got %q", err.Error())
	}
}
