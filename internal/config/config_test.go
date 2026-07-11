package config

import "testing"

func TestDefaultPoliciesAreApplicationOwned(t *testing.T) {
	policies := Default().Policies
	want := []string{"Proxy", "Direct", "Final"}
	if len(policies) != len(want) {
		t.Fatalf("policy count = %d, want %d", len(policies), len(want))
	}
	for i, name := range want {
		if policies[i].Name != name || !policies[i].Enabled {
			t.Fatalf("policy[%d] = %+v, want enabled %q", i, policies[i], name)
		}
	}
	for _, policy := range policies {
		if !policy.System {
			t.Fatalf("default policy is not system-owned: %+v", policy)
		}
	}
	if policies[0].Kind != PolicySelector || policies[1].Kind != PolicyDirect || policies[2].Kind != PolicyFinal {
		t.Fatalf("unexpected policy kinds: %+v", policies)
	}
}
