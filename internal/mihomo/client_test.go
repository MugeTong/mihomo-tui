package mihomo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientSendsBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization header = %q, want %q", got, "Bearer secret")
		}
		writeJSON(t, w, Version{Version: "test"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Health(); err != nil {
		t.Fatal(err)
	}
}

func TestProxyGroupsBuildsGroupsWithProxyDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxies" {
			t.Fatalf("path = %q, want /proxies", r.URL.Path)
		}

		writeJSON(t, w, map[string]any{
			"proxies": map[string]any{
				"GLOBAL": map[string]any{
					"name": "GLOBAL",
					"type": "Selector",
					"now":  "Tokyo",
					"all":  []string{"Tokyo", "Hong Kong"},
				},
				"Tokyo": map[string]any{
					"name": "Tokyo",
					"type": "Shadowsocks",
					"udp":  true,
					"history": []map[string]any{
						{"time": "2026-06-27T00:00:00Z", "delay": 92},
					},
				},
				"Hong Kong": map[string]any{
					"name": "Hong Kong",
					"type": "Trojan",
					"udp":  false,
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}

	groups, err := client.ProxyGroups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if groups[0].Name != "GLOBAL" || groups[0].Now != "Tokyo" {
		t.Fatalf("group = %+v", groups[0])
	}
	if got := groups[0].Proxies[0].Delay; got != 92 {
		t.Fatalf("delay = %d, want 92", got)
	}
	if got := groups[0].Proxies[1].Delay; got != -1 {
		t.Fatalf("missing delay = %d, want -1", got)
	}
}

func TestSelectProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/proxies/GLOBAL" {
			t.Fatalf("path = %q, want /proxies/GLOBAL", r.URL.Path)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["name"] != "Tokyo" {
			t.Fatalf("selected proxy = %q, want Tokyo", body["name"])
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.SelectProxy("GLOBAL", "Tokyo"); err != nil {
		t.Fatal(err)
	}
}

func TestTestProxyDelay(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxies/Tokyo/delay" {
			t.Fatalf("path = %q, want /proxies/Tokyo/delay", r.URL.Path)
		}
		if got := r.URL.Query().Get("timeout"); got != "1500" {
			t.Fatalf("timeout = %q, want 1500", got)
		}
		if got := r.URL.Query().Get("url"); got == "" {
			t.Fatal("url query is empty")
		}
		writeJSON(t, w, map[string]int{"delay": 88})
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}

	delay, err := client.TestProxyDelay("Tokyo", 1500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if delay != 88 {
		t.Fatalf("delay = %d, want 88", delay)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}
