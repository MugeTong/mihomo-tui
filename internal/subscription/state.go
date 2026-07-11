package subscription

import (
	"fmt"
	"strings"
)

const CurrentStateVersion = 2

type State struct {
	Version    int               `json:"version"`
	Sources    []Source          `json:"sources"`
	Nodes      []Node            `json:"nodes"`
	Selections []PolicySelection `json:"policy_selections"`
}

type ReconcileReport struct{ Issues []string }

func NewState() State { return State{Version: CurrentStateVersion} }

// AddImport merges an import into the global node pool. Display names are the
// user-facing identity: a later node with the same name replaces the old one.
func (s *State) AddImport(source *Source, result ImportResult) (MergeReport, error) {
	report := MergeReport{Duplicates: result.Duplicates}
	if source != nil {
		if source.Type != SourceURL || strings.TrimSpace(source.Location) == "" {
			return report, fmt.Errorf("invalid source metadata")
		}
		location := strings.TrimSpace(source.Location)
		found := false
		for index := range s.Sources {
			if s.Sources[index].Location == location {
				s.Sources[index] = *source
				found = true
				break
			}
		}
		if !found {
			s.Sources = append(s.Sources, *source)
		}
	}

	knownIDs := make(map[string]struct{}, len(s.Nodes)+len(result.Nodes))
	usedNames := make(map[string]struct{}, len(s.Nodes)+len(result.Nodes))
	for _, node := range s.Nodes {
		knownIDs[node.ID] = struct{}{}
		usedNames[node.Name] = struct{}{}
	}
	for _, node := range result.Nodes {
		if err := validateNode(node); err != nil {
			return report, fmt.Errorf("invalid imported node %s: %w", safeNodeName(node.Name), err)
		}
		if _, exists := knownIDs[node.ID]; exists {
			report.Duplicates++
			continue
		}
		base := node.Name
		if _, exists := usedNames[base]; exists {
			for number := 2; ; number++ {
				candidate := fmt.Sprintf("%s (%d)", base, number)
				if _, exists := usedNames[candidate]; !exists {
					node.Name = candidate
					break
				}
			}
			report.Renamed++
		}
		usedNames[node.Name] = struct{}{}
		knownIDs[node.ID] = struct{}{}
		s.Nodes = append(s.Nodes, node)
		report.Added++
	}
	return report, nil
}

func (s *State) Reconcile() ReconcileReport {
	report := ReconcileReport{}
	if s.Version != CurrentStateVersion {
		*s = NewState()
		report.Issues = append(report.Issues, "discarded incompatible subscription state")
		return report
	}

	seenSources := make(map[string]struct{}, len(s.Sources))
	sources := make([]Source, 0, len(s.Sources))
	for _, source := range s.Sources {
		source.Location = strings.TrimSpace(source.Location)
		if source.Type != SourceURL || source.Location == "" {
			report.Issues = append(report.Issues, "removed invalid source")
			continue
		}
		if _, exists := seenSources[source.Location]; exists {
			report.Issues = append(report.Issues, "removed duplicate source")
			continue
		}
		seenSources[source.Location] = struct{}{}
		sources = append(sources, source)
	}
	s.Sources = sources

	oldToNewID := make(map[string]string, len(s.Nodes))
	knownIDs := make(map[string]struct{}, len(s.Nodes))
	usedNames := make(map[string]struct{}, len(s.Nodes))
	nodes := make([]Node, 0, len(s.Nodes))
	for _, node := range s.Nodes {
		oldID := node.ID
		if err := validateNode(node); err != nil {
			report.Issues = append(report.Issues, "removed invalid node "+safeNodeName(node.Name))
			continue
		}
		id, err := stableNodeID(node)
		if err != nil {
			report.Issues = append(report.Issues, "removed node with unsupported options "+safeNodeName(node.Name))
			continue
		}
		node.ID = id
		oldToNewID[oldID] = id
		oldToNewID[id] = id
		if _, exists := knownIDs[node.ID]; exists {
			report.Issues = append(report.Issues, "removed duplicate node "+safeNodeName(node.Name))
			continue
		}
		if _, exists := usedNames[node.Name]; exists {
			base := node.Name
			for number := 2; ; number++ {
				candidate := fmt.Sprintf("%s (%d)", base, number)
				if _, exists := usedNames[candidate]; !exists {
					node.Name = candidate
					break
				}
			}
			report.Issues = append(report.Issues, "renamed duplicate node "+safeNodeName(base))
		}
		knownIDs[node.ID] = struct{}{}
		usedNames[node.Name] = struct{}{}
		nodes = append(nodes, node)
	}
	s.Nodes = nodes

	validIDs := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		validIDs[node.ID] = struct{}{}
	}
	selections := make([]PolicySelection, 0, len(s.Selections))
	seenPolicies := make(map[string]struct{}, len(s.Selections))
	for _, selection := range s.Selections {
		selection.NodeID = oldToNewID[selection.NodeID]
		if strings.TrimSpace(selection.Policy) == "" {
			continue
		}
		if _, exists := validIDs[selection.NodeID]; !exists {
			continue
		}
		if _, exists := seenPolicies[selection.Policy]; exists {
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

func safeNodeName(name string) string {
	if name = strings.TrimSpace(name); name != "" {
		return name
	}
	return "(unnamed)"
}
