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

	cases := []struct {
		path    string
		minRows int
	}{
		{"/mgmt/tm/ltm/virtual", 1},
		{"/mgmt/tm/ltm/pool", 1},
		{"/mgmt/tm/ltm/policy", 1},
		{"/mgmt/tm/asm/policies", 1},
		{"/mgmt/tm/ltm/pool/~Common~pool_web/members", 1},
	}
	for _, c := range cases {
		resp, err := http.Get(srv.URL + c.path)
		if err != nil {
			t.Errorf("GET %s: %v", c.path, err)
			continue
		}
		if resp.StatusCode != 200 {
			t.Errorf("GET %s: status %d", c.path, resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
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
}
