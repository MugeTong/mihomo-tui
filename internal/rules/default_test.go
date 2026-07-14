package rules

import "testing"

func TestEmbeddedDefaultsParse(t *testing.T) {
	rules, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) < 40 {
		t.Fatalf("default rule count = %d, want at least 40", len(rules))
	}
	last := rules[len(rules)-1]
	if last.Type != "MATCH" || last.Policy != "Final" {
		t.Fatalf("last rule = %+v, want MATCH,Final", last)
	}
}

func TestEmbeddedDefaultsUseKnownPolicies(t *testing.T) {
	rules, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range rules {
		if rule.Policy != "Proxy" && rule.Policy != "DIRECT" && rule.Policy != "Final" {
			t.Fatalf("unknown policy in embedded defaults: %+v", rule)
		}
	}
}

func TestEmbeddedDefaultsRouteNexusDirectly(t *testing.T) {
	rules, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range rules {
		if rule.Type == "DOMAIN-KEYWORD" && rule.Value == "nexus" {
			if rule.Policy != "DIRECT" {
				t.Fatalf("nexus policy = %q, want DIRECT", rule.Policy)
			}
			return
		}
	}
	t.Fatal("embedded defaults do not contain the nexus direct rule")
}

func TestEmbeddedDefaultsRouteEducationDomainsDirectly(t *testing.T) {
	rules, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range rules {
		if rule.Type == "DOMAIN-SUFFIX" && rule.Value == "edu.cn" {
			if rule.Policy != "DIRECT" {
				t.Fatalf("edu.cn policy = %q, want DIRECT", rule.Policy)
			}
			return
		}
	}
	t.Fatal("embedded defaults do not contain the edu.cn direct rule")
}
