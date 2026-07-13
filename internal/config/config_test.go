package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultPathsUseFixedHomeLayout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got, want := Default().ConfigPath, filepath.Join(home, ".config", "mihomo-tui", "config.yaml"); got != want {
		t.Fatalf("runtime config path = %q, want %q", got, want)
	}
	if got, err := Path(); err != nil || got != filepath.Join(home, ".config", "mihomo-tui", "config.json") {
		t.Fatalf("app config path = %q, err = %v", got, err)
	}
}

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
	if policies[0].Kind != PolicySelector || policies[1].Kind != PolicyDirect || policies[2].Kind != PolicyFinal {
		t.Fatalf("unexpected policy kinds: %+v", policies)
	}
}
