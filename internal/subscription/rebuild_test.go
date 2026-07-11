package subscription

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRebuildDerivesNodesFromURLAndURI(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := "trojan://password@url.example.test:443#URL-Node-A\ntrojan://password@url.example.test:443#URL-Node-B"
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	sources := []Source{
		{Type: SourceURL, Location: "https://subscription.example.test/token"},
		{Type: SourceURI, Location: "vless://uuid@uri.example.test:443?security=tls#URI-Node"},
		{Type: SourceURI, Location: "trojan://password@url.example.test:443#Cross-Source-Alias"},
	}
	nodes, report, err := Rebuild(sources, Fetcher{Client: client})
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 3 || report.Added != 3 || report.Duplicates != 1 || report.Skipped != 0 {
		t.Fatalf("nodes=%+v report=%+v", nodes, report)
	}
}

func TestRebuildReturnsNoPartialPoolWhenSourceFails(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Status: "503 Service Unavailable", Body: io.NopCloser(strings.NewReader("unavailable")), Header: make(http.Header)}, nil
	})}
	sources := []Source{
		{Type: SourceURI, Location: "trojan://password@ok.example.test:443#OK"},
		{Type: SourceURL, Location: "https://subscription.example.test/token"},
	}
	nodes, _, err := Rebuild(sources, Fetcher{Client: client})
	if err == nil {
		t.Fatal("failed source unexpectedly rebuilt")
	}
	if nodes != nil {
		t.Fatalf("partial nodes returned: %+v", nodes)
	}
}
