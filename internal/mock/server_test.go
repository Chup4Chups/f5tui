package mock

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestMockEndpoints(t *testing.T) {
	srv := Start()
	defer srv.Close()

	// Endpoints that return a `{ "items": [...] }` envelope.
	envelopes := []struct {
		path    string
		minRows int
	}{
		{"/mgmt/tm/ltm/virtual", 1},
		{"/mgmt/tm/ltm/pool", 1},
		{"/mgmt/tm/ltm/policy", 1},
		{"/mgmt/tm/asm/policies", 1},
		{"/mgmt/tm/ltm/pool/~Common~pool_web/members", 1},
		{"/mgmt/tm/asm/policies/abc123/urls", 1},
		{"/mgmt/tm/asm/policies/abc123/parameters", 1},
	}
	for _, c := range envelopes {
		resp, err := http.Get(srv.URL + c.path)
		if err != nil {
			t.Errorf("GET %s: %v", c.path, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("GET %s: status %d", c.path, resp.StatusCode)
			continue
		}
		var env struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(body, &env); err != nil {
			t.Errorf("GET %s: invalid json: %v", c.path, err)
			continue
		}
		if len(env.Items) < c.minRows {
			t.Errorf("GET %s: want >=%d items, got %d", c.path, c.minRows, len(env.Items))
		}
	}

	// Endpoints that return a single object with at least a "name" or "id" field.
	singles := []struct {
		path string
		key  string
	}{
		{"/mgmt/tm/ltm/virtual/~Common~vs_web_http", "name"},
		{"/mgmt/tm/ltm/pool/~Common~pool_web", "name"},
		{"/mgmt/tm/ltm/policy/~Common~policy_host_routing", "name"},
		{"/mgmt/tm/asm/policies/abc123", "id"},
	}
	for _, c := range singles {
		resp, err := http.Get(srv.URL + c.path)
		if err != nil {
			t.Errorf("GET %s: %v", c.path, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("GET %s: status %d", c.path, resp.StatusCode)
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(body, &obj); err != nil {
			t.Errorf("GET %s: invalid json: %v", c.path, err)
			continue
		}
		if _, ok := obj[c.key]; !ok {
			t.Errorf("GET %s: missing key %q in %v", c.path, c.key, obj)
		}
	}

	// 404 for unknown detail
	resp, _ := http.Get(srv.URL + "/mgmt/tm/ltm/virtual/~Common~does_not_exist")
	if resp.StatusCode != 404 {
		t.Errorf("unknown VS: want 404, got %d", resp.StatusCode)
	}
}
