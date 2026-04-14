package ui

import "testing"

func TestMatchPartition(t *testing.T) {
	cases := []struct {
		current string
		item    string
		want    bool
	}{
		{"", "Common", true},
		{"", "Tenant_A", true},
		{"Common", "Common", true},
		{"common", "Common", true}, // case-insensitive
		{"Common", "Tenant_A", false},
	}
	for _, c := range cases {
		a := &App{partition: c.current}
		if got := a.matchPartition(c.item); got != c.want {
			t.Errorf("partition=%q item=%q: got %v, want %v", c.current, c.item, got, c.want)
		}
	}
}

func TestMatchFilter(t *testing.T) {
	cases := []struct {
		filter string
		fields []string
		want   bool
	}{
		{"", []string{"anything"}, true},
		{"web", []string{"vs_web_http", "/Common/vs_web_http"}, true},
		{"WEB", []string{"vs_web_http"}, true}, // case-insensitive
		{"api", []string{"vs_web_http", "/Common/pool_web"}, false},
		{"10.0", []string{"foo", "10.0.0.1:80"}, true},
	}
	for _, c := range cases {
		a := &App{filter: c.filter}
		if got := a.matchFilter(c.fields...); got != c.want {
			t.Errorf("filter=%q fields=%v: got %v, want %v", c.filter, c.fields, got, c.want)
		}
	}
}
