package subscription

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestFetcherImportsAndAutoDetectsYAML(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("User-Agent") != "mihomo-tui" {
			t.Fatalf("User-Agent = %q", request.Header.Get("User-Agent"))
		}
		body := `proxies:
  - {name: Tokyo, type: trojan, server: jp.example.test, port: 443, password: test}
rules:
  - MATCH,ProviderRule
`
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	fetcher := Fetcher{Client: client}
	result, err := fetcher.Import("https://subscription.example.test/token", "provider")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Nodes) != 1 || result.Nodes[0].Name != "Tokyo" {
		t.Fatalf("result = %+v", result)
	}
}

func TestFetcherRejectsPrivateAndCredentialedURLs(t *testing.T) {
	fetcher := Fetcher{Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("blocked URL reached transport")
		return nil, nil
	})}}
	for _, location := range []string{
		"http://127.0.0.1/sub",
		"http://169.254.169.254/latest/meta-data",
		"https://user:password@example.test/sub",
		"file:///tmp/subscription",
	} {
		if _, err := fetcher.Import(location, "provider"); err == nil {
			t.Fatalf("unsafe URL %q unexpectedly accepted", location)
		}
	}
}
