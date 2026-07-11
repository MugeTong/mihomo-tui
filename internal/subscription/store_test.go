package subscription

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreRoundTripUsesPrivateAtomicFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	store := Store{Path: path}
	node := Node{Name: "Tokyo", Protocol: ProtocolShadowsocks, Server: "jp.example.test", Port: 443, Options: map[string]any{"password": "secret"}}
	state := State{
		Version: CurrentStateVersion,
		Sources: []Source{{Type: SourceURL, Location: "https://sub.example.test/token"}},
		Nodes:   []Node{node},
	}
	id, err := stableNodeID(node)
	if err != nil {
		t.Fatal(err)
	}
	state.Nodes[0].ID = id

	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("state permissions = %o, want 600", got)
	}

	loaded, report, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 0 || len(loaded.Nodes) != 0 || len(loaded.Sources) != 1 {
		t.Fatalf("loaded state = %+v, report = %+v", loaded, report)
	}
	persisted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(persisted), "secret") || strings.Contains(string(persisted), "\"nodes\"") {
		t.Fatal("derived nodes leaked into state file")
	}
}

func TestDefaultStatePathFollowsXDGConfigHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", home)
	got, err := DefaultStatePath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "mihomo-tui", "state.json"); got != want {
		t.Fatalf("state path = %q, want %q", got, want)
	}
}

func TestStoreDoesNotOverwriteMalformedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	original := []byte("{not-json")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	store := Store{Path: path}
	if _, _, err := store.Load(); err == nil {
		t.Fatal("malformed state unexpectedly loaded")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatal("malformed state was overwritten")
	}
}
