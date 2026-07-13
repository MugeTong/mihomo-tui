package subscription

import "fmt"

type RebuildReport struct {
	Added      int
	Duplicates int
	Renamed    int
	Skipped    int
}

// Rebuild derives a fresh node pool from every persisted source.
func Rebuild(sources []Source, fetcher Fetcher) ([]Node, RebuildReport, error) {
	defer fetcher.closeIdleConnections()
	nodes := []Node{}
	report := RebuildReport{}
	for index, source := range sources {
		var result ImportResult
		var err error
		switch source.Type {
		case SourceURL:
			result, err = fetcher.Import(source.Location)
		case SourceURI:
			result, err = ImportShareLinks([]byte(source.Location))
		default:
			err = fmt.Errorf("unsupported source type %q", source.Type)
		}
		if err != nil {
			return nil, report, fmt.Errorf("rebuild source %d: %w", index+1, err)
		}
		var merged MergeReport
		nodes, merged, err = mergeNodes(nodes, result)
		if err != nil {
			return nil, report, err
		}
		report.Added += merged.Added
		report.Duplicates += merged.Duplicates
		report.Renamed += merged.Renamed
		report.Skipped += len(result.Issues)
	}
	return nodes, report, nil
}
