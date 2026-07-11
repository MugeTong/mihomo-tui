package subscription

import (
	"fmt"
	"strings"
)

const CurrentStateVersion = 1

type State struct {
	Version    int               `json:"version"`
	Sources    []Source          `json:"sources"`
	Nodes      []Node            `json:"nodes"`
	Links      []SourceNode      `json:"source_nodes"`
	Selections []PolicySelection `json:"policy_selections"`
}

type ReconcileReport struct {
	Issues []string
}

func NewState() State {
	return State{Version: CurrentStateVersion}
}

// Reconcile validates references, recalculates node IDs, merges duplicate
// nodes, and removes dangling links and selections.
func (s *State) Reconcile() ReconcileReport {
	report := ReconcileReport{}
	if s.Version == 0 {
		s.Version = CurrentStateVersion
	}
	if s.Version != CurrentStateVersion {
		report.Issues = append(report.Issues, fmt.Sprintf("unsupported state version %d", s.Version))
		return report
	}

	validSources := make(map[string]struct{}, len(s.Sources))
	sources := make([]Source, 0, len(s.Sources))
	for _, source := range s.Sources {
		source.ID = strings.TrimSpace(source.ID)
		source.Name = strings.TrimSpace(source.Name)
		if source.ID == "" || source.Name == "" || !validSourceType(source.Type) {
			report.Issues = append(report.Issues, "removed invalid source metadata")
			continue
		}
		if _, duplicate := validSources[source.ID]; duplicate {
			report.Issues = append(report.Issues, "removed duplicate source "+source.ID)
			continue
		}
		validSources[source.ID] = struct{}{}
		sources = append(sources, source)
	}
	s.Sources = sources

	oldToNewID := make(map[string]string, len(s.Nodes))
	validNodes := make(map[string]struct{}, len(s.Nodes))
	nodes := make([]Node, 0, len(s.Nodes))
	for _, node := range s.Nodes {
		oldID := node.ID
		if err := validateNode(node); err != nil {
			report.Issues = append(report.Issues, "removed invalid node "+safeNodeName(node.Name)+": "+err.Error())
			continue
		}
		calculatedID, err := stableNodeID(node)
		if err != nil {
			report.Issues = append(report.Issues, "removed node with unsupported options "+safeNodeName(node.Name))
			continue
		}
		node.ID = calculatedID
		oldToNewID[oldID] = calculatedID
		oldToNewID[calculatedID] = calculatedID
		if _, duplicate := validNodes[calculatedID]; duplicate {
			report.Issues = append(report.Issues, "merged duplicate node "+safeNodeName(node.Name))
			continue
		}
		validNodes[calculatedID] = struct{}{}
		nodes = append(nodes, node)
	}
	s.Nodes = nodes

	linkKeys := make(map[string]struct{}, len(s.Links))
	links := make([]SourceNode, 0, len(s.Links))
	for _, link := range s.Links {
		remappedID, knownID := oldToNewID[link.NodeID]
		_, knownSource := validSources[link.SourceID]
		if !knownSource || !knownID {
			report.Issues = append(report.Issues, "removed dangling source-node link")
			continue
		}
		link.NodeID = remappedID
		key := link.SourceID + "\x00" + link.NodeID
		if _, duplicate := linkKeys[key]; duplicate {
			report.Issues = append(report.Issues, "removed duplicate source-node link")
			continue
		}
		linkKeys[key] = struct{}{}
		links = append(links, link)
	}
	s.Links = links

	selections := make([]PolicySelection, 0, len(s.Selections))
	seenPolicies := make(map[string]struct{}, len(s.Selections))
	for _, selection := range s.Selections {
		selection.NodeID = oldToNewID[selection.NodeID]
		if strings.TrimSpace(selection.Policy) == "" {
			report.Issues = append(report.Issues, "removed unnamed policy selection")
			continue
		}
		if _, exists := validNodes[selection.NodeID]; !exists {
			report.Issues = append(report.Issues, "removed policy selection with missing node")
			continue
		}
		if _, duplicate := seenPolicies[selection.Policy]; duplicate {
			report.Issues = append(report.Issues, "removed duplicate policy selection "+selection.Policy)
			continue
		}
		seenPolicies[selection.Policy] = struct{}{}
		selections = append(selections, selection)
	}
	s.Selections = selections
	return report
}

func validateNode(node Node) error {
	if strings.TrimSpace(node.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if !supportedProtocol(node.Protocol) {
		return fmt.Errorf("unsupported protocol %q", node.Protocol)
	}
	if strings.TrimSpace(node.Server) == "" {
		return fmt.Errorf("server is required")
	}
	if node.Port < 1 || node.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func validSourceType(sourceType SourceType) bool {
	switch sourceType {
	case SourceURL, SourceFile, SourcePaste, SourceManual:
		return true
	default:
		return false
	}
}

func safeNodeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "(unnamed)"
	}
	return name
}
