package subscription

import (
	"fmt"
	"strings"
)

const CurrentStateVersion = 3

type State struct {
	Version    int               `json:"version"`
	Sources    []Source          `json:"sources"`
	Selections []PolicySelection `json:"policy_selections"`
	Nodes      []Node            `json:"-"` // Derived from Sources; never persisted.
}

type ReconcileReport struct{ Issues []string }

func NewState() State { return State{Version: CurrentStateVersion} }

func (s *State) AddSource(source Source) bool {
	source.Location = strings.TrimSpace(source.Location)
	for _, existing := range s.Sources {
		if existing.Type == source.Type && existing.Location == source.Location {
			return false
		}
	}
	s.Sources = append(s.Sources, source)
	return true
}

func (s *State) Reconcile() ReconcileReport {
	report := ReconcileReport{}
	if s.Version != CurrentStateVersion {
		*s = NewState()
		report.Issues = append(report.Issues, "discarded incompatible subscription state")
		return report
	}
	seen := make(map[string]struct{}, len(s.Sources))
	sources := make([]Source, 0, len(s.Sources))
	for _, source := range s.Sources {
		source.Location = strings.TrimSpace(source.Location)
		if (source.Type != SourceURL && source.Type != SourceURI) || source.Location == "" {
			report.Issues = append(report.Issues, "removed invalid source")
			continue
		}
		key := string(source.Type) + "\x00" + source.Location
		if _, duplicate := seen[key]; duplicate {
			report.Issues = append(report.Issues, "removed duplicate source")
			continue
		}
		seen[key] = struct{}{}
		sources = append(sources, source)
	}
	s.Sources = sources
	return report
}

func mergeNodes(existing []Node, result ImportResult) ([]Node, MergeReport, error) {
	report := MergeReport{Duplicates: result.Duplicates}
	// IDs already present before this source are cross-source duplicates. Do
	// not add current-source IDs here: one provider may intentionally expose
	// multiple differently named aliases for the same connection.
	knownIDs := make(map[string]struct{}, len(existing))
	usedNames := make(map[string]struct{}, len(existing)+len(result.Nodes))
	for _, node := range existing {
		knownIDs[node.ID] = struct{}{}
		usedNames[node.Name] = struct{}{}
	}
	for _, node := range result.Nodes {
		if err := validateNode(node); err != nil {
			return nil, report, fmt.Errorf("invalid imported node %s: %w", safeNodeName(node.Name), err)
		}
		if _, duplicate := knownIDs[node.ID]; duplicate {
			report.Duplicates++
			continue
		}
		if _, collision := usedNames[node.Name]; collision {
			base := node.Name
			for number := 2; ; number++ {
				candidate := fmt.Sprintf("%s (%d)", base, number)
				if _, used := usedNames[candidate]; !used {
					node.Name = candidate
					break
				}
			}
			report.Renamed++
		}
		usedNames[node.Name] = struct{}{}
		existing = append(existing, node)
		report.Added++
	}
	return existing, report, nil
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
