package mock

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
	"strings"
)

//go:embed fixtures/virtual.json
var virtualJSON []byte

//go:embed fixtures/pool.json
var poolJSON []byte

//go:embed fixtures/pool_members.json
var poolMembersJSON []byte

//go:embed fixtures/policy.json
var policyJSON []byte

//go:embed fixtures/asm_policies.json
var asmJSON []byte

func Start() *httptest.Server {
	mux := http.NewServeMux()
	serve := func(path string, body []byte) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		})
	}
	serve("/mgmt/tm/ltm/virtual", virtualJSON)
	serve("/mgmt/tm/ltm/pool", poolJSON)
	serve("/mgmt/tm/ltm/policy", policyJSON)
	serve("/mgmt/tm/asm/policies", asmJSON)

	mux.HandleFunc("/mgmt/tm/ltm/pool/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/members") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(poolMembersJSON)
			return
		}
		http.NotFound(w, r)
	})

	return httptest.NewServer(mux)
}
