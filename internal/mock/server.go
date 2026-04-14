package mock

import (
	_ "embed"
	"encoding/json"
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

//go:embed fixtures/virtual_detail.json
var virtualDetailRaw []byte

//go:embed fixtures/pool_detail.json
var poolDetailRaw []byte

//go:embed fixtures/policy_detail.json
var policyDetailRaw []byte

//go:embed fixtures/asm_detail.json
var asmDetailRaw []byte

//go:embed fixtures/asm_urls.json
var asmURLsRaw []byte

//go:embed fixtures/asm_parameters.json
var asmParamsRaw []byte

func parseMap(raw []byte) map[string]json.RawMessage {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	return m
}

// decodeLTMName turns a path segment like "~Common~vs_web_http" into "/Common/vs_web_http".
func decodeLTMName(seg string) string {
	return strings.ReplaceAll(seg, "~", "/")
}

// pathTail returns the part of urlPath after prefix, with any trailing "/members" stripped.
func pathTail(urlPath, prefix string) (name, sub string) {
	rest := strings.TrimPrefix(urlPath, prefix)
	rest = strings.TrimSuffix(rest, "/")
	if i := strings.Index(rest, "/"); i >= 0 {
		return rest[:i], rest[i+1:]
	}
	return rest, ""
}

func Start() *httptest.Server {
	virtualDetails := parseMap(virtualDetailRaw)
	poolDetails := parseMap(poolDetailRaw)
	policyDetails := parseMap(policyDetailRaw)
	asmDetails := parseMap(asmDetailRaw)
	asmURLs := parseMap(asmURLsRaw)
	asmParams := parseMap(asmParamsRaw)

	mux := http.NewServeMux()
	serve := func(path string, body []byte) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		})
	}
	writeJSON := func(w http.ResponseWriter, raw json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}

	// LIST endpoints
	serve("/mgmt/tm/ltm/virtual", virtualJSON)
	serve("/mgmt/tm/ltm/pool", poolJSON)
	serve("/mgmt/tm/ltm/policy", policyJSON)
	serve("/mgmt/tm/asm/policies", asmJSON)

	// VIRTUAL detail: /mgmt/tm/ltm/virtual/~Common~vs_web_http
	mux.HandleFunc("/mgmt/tm/ltm/virtual/", func(w http.ResponseWriter, r *http.Request) {
		name, _ := pathTail(r.URL.Path, "/mgmt/tm/ltm/virtual/")
		key := decodeLTMName(name)
		if body, ok := virtualDetails[key]; ok {
			writeJSON(w, body)
			return
		}
		http.NotFound(w, r)
	})

	// POOL detail + members: /mgmt/tm/ltm/pool/~Common~pool_web[/members]
	mux.HandleFunc("/mgmt/tm/ltm/pool/", func(w http.ResponseWriter, r *http.Request) {
		name, sub := pathTail(r.URL.Path, "/mgmt/tm/ltm/pool/")
		if sub == "members" {
			writeJSON(w, poolMembersJSON)
			return
		}
		key := decodeLTMName(name)
		if body, ok := poolDetails[key]; ok {
			writeJSON(w, body)
			return
		}
		http.NotFound(w, r)
	})

	// LTM POLICY detail: /mgmt/tm/ltm/policy/~Common~policy_host_routing
	mux.HandleFunc("/mgmt/tm/ltm/policy/", func(w http.ResponseWriter, r *http.Request) {
		name, _ := pathTail(r.URL.Path, "/mgmt/tm/ltm/policy/")
		key := decodeLTMName(name)
		if body, ok := policyDetails[key]; ok {
			writeJSON(w, body)
			return
		}
		http.NotFound(w, r)
	})

	// ASM detail / urls / parameters: /mgmt/tm/asm/policies/{id}[/urls|/parameters]
	mux.HandleFunc("/mgmt/tm/asm/policies/", func(w http.ResponseWriter, r *http.Request) {
		id, sub := pathTail(r.URL.Path, "/mgmt/tm/asm/policies/")
		switch sub {
		case "":
			if body, ok := asmDetails[id]; ok {
				writeJSON(w, body)
				return
			}
		case "urls":
			if body, ok := asmURLs[id]; ok {
				// ASM sub-endpoints use {"items":[...]} envelope.
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"items":`))
				w.Write(body)
				w.Write([]byte(`}`))
				return
			}
		case "parameters":
			if body, ok := asmParams[id]; ok {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"items":`))
				w.Write(body)
				w.Write([]byte(`}`))
				return
			}
		}
		http.NotFound(w, r)
	})

	return httptest.NewServer(mux)
}
