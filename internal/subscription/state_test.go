package subscription

import "testing"

func TestAddImportDeduplicatesIDAndRenamesNameCollision(t *testing.T) {
	old := Node{ID: "old", Name: "Tokyo", Protocol: ProtocolTrojan, Server: "old.example.test", Port: 443, Options: map[string]any{"password": "old"}}
	next := Node{Name: "Tokyo", Protocol: ProtocolVLESS, Server: "new.example.test", Port: 8443, Options: map[string]any{"uuid": "new"}}
	var err error
	next.ID, err = stableNodeID(next)
	if err != nil {
		t.Fatal(err)
	}
	state := State{Version: CurrentStateVersion, Nodes: []Node{old}}
	report, err := state.AddImport(nil, ImportResult{Nodes: []Node{next, old}})
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Nodes) != 2 || state.Nodes[1].Name != "Tokyo (2)" {
		t.Fatalf("nodes = %+v", state.Nodes)
	}
	if report.Added != 1 || report.Duplicates != 1 || report.Renamed != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestAddImportDeduplicatesSourceURL(t *testing.T) {
	state := NewState()
	source := Source{Type: SourceURL, Location: "https://sub.example.test/token"}
	if _, err := state.AddImport(&source, ImportResult{}); err != nil {
		t.Fatal(err)
	}
	if _, err := state.AddImport(&source, ImportResult{}); err != nil {
		t.Fatal(err)
	}
	if len(state.Sources) != 1 {
		t.Fatalf("sources = %+v", state.Sources)
	}
}

func TestReconcileDiscardsOldStateVersion(t *testing.T) {
	state := State{Version: 1, Nodes: []Node{{Name: "Old"}}}
	report := state.Reconcile()
	if state.Version != CurrentStateVersion || len(state.Nodes) != 0 || len(report.Issues) != 1 {
		t.Fatalf("state=%+v report=%+v", state, report)
	}
}
