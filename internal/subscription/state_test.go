package subscription

import "testing"

func TestReconcileRepairsIDsAndDropsDanglingReferences(t *testing.T) {
	node := Node{
		ID:       "tampered-id",
		Name:     "Tokyo",
		Protocol: ProtocolTrojan,
		Server:   "jp.example.test",
		Port:     443,
		Options:  map[string]any{"password": "secret"},
	}
	state := State{
		Version: CurrentStateVersion,
		Sources: []Source{{ID: "source-a", Name: "Provider", Type: SourceURL, Enabled: true}},
		Nodes:   []Node{node},
		Links: []SourceNode{
			{SourceID: "source-a", NodeID: "tampered-id", Alias: "Tokyo"},
			{SourceID: "missing", NodeID: "tampered-id"},
		},
		Selections: []PolicySelection{
			{Policy: "Proxy", NodeID: "tampered-id"},
			{Policy: "Missing", NodeID: "not-found"},
		},
	}

	report := state.Reconcile()
	if len(report.Issues) == 0 {
		t.Fatal("expected repair report")
	}
	if len(state.Nodes) != 1 || state.Nodes[0].ID == "tampered-id" {
		t.Fatalf("node ID was not repaired: %+v", state.Nodes)
	}
	wantID := state.Nodes[0].ID
	if len(state.Links) != 1 || state.Links[0].NodeID != wantID {
		t.Fatalf("links were not reconciled: %+v", state.Links)
	}
	if len(state.Selections) != 1 || state.Selections[0].NodeID != wantID {
		t.Fatalf("selections were not reconciled: %+v", state.Selections)
	}
}

func TestReconcileMergesDuplicateNodes(t *testing.T) {
	node := Node{Name: "Tokyo", Protocol: ProtocolVLESS, Server: "edge.example.test", Port: 443, Options: map[string]any{"uuid": "test"}}
	node.ID = "first"
	duplicate := node
	duplicate.ID = "second"
	duplicate.Name = "Renamed"
	state := State{
		Version: CurrentStateVersion,
		Sources: []Source{{ID: "a", Name: "A", Type: SourceShare}},
		Nodes:   []Node{node, duplicate},
		Links:   []SourceNode{{SourceID: "a", NodeID: "first"}, {SourceID: "a", NodeID: "second"}},
	}

	state.Reconcile()
	if len(state.Nodes) != 1 || len(state.Links) != 1 {
		t.Fatalf("duplicates were not merged: nodes=%+v links=%+v", state.Nodes, state.Links)
	}
}
