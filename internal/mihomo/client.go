package mihomo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const defaultDelayTestURL = "https://www.gstatic.com/generate_204"

type Client struct {
	baseURL    *url.URL
	secret     string
	httpClient *http.Client
}

func NewClient(baseURL, secret string) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://127.0.0.1:9090"
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse mihomo controller URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("mihomo controller URL must include scheme and host")
	}

	return &Client{
		baseURL: parsed,
		secret:  secret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (c *Client) Health() error {
	_, err := c.Version()
	return err
}

func (c *Client) Version() (Version, error) {
	var version Version
	if err := c.do(http.MethodGet, "/version", nil, &version); err != nil {
		return Version{}, err
	}
	return version, nil
}

func (c *Client) ProxyGroups() ([]ProxyGroup, error) {
	var response proxyResponse
	if err := c.do(http.MethodGet, "/proxies", nil, &response); err != nil {
		return nil, err
	}

	groups := make([]ProxyGroup, 0)
	for name, item := range response.Proxies {
		if len(item.All) == 0 {
			continue
		}

		group := ProxyGroup{
			Name: name,
			Type: item.Type,
			Now:  item.Now,
			All:  append([]string(nil), item.All...),
		}

		for _, proxyName := range item.All {
			proxyItem, ok := response.Proxies[proxyName]
			if !ok {
				group.Proxies = append(group.Proxies, Proxy{Name: proxyName, Delay: -1})
				continue
			}

			group.Proxies = append(group.Proxies, Proxy{
				Name:  proxyName,
				Type:  proxyItem.Type,
				UDP:   proxyItem.UDP,
				Delay: latestDelay(proxyItem.History),
			})
		}

		groups = append(groups, group)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})

	return groups, nil
}

func (c *Client) SelectProxy(groupName, proxyName string) error {
	body := map[string]string{"name": proxyName}
	return c.do(http.MethodPut, "/proxies/"+url.PathEscape(groupName), body, nil)
}

func (c *Client) TestProxyDelay(proxyName string, timeout time.Duration) (int, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	values := url.Values{}
	values.Set("url", defaultDelayTestURL)
	values.Set("timeout", fmt.Sprintf("%d", timeout.Milliseconds()))

	var response delayResponse
	endpoint := "/proxies/" + url.PathEscape(proxyName) + "/delay?" + values.Encode()
	if err := c.do(http.MethodGet, endpoint, nil, &response); err != nil {
		return 0, err
	}
	return response.Delay, nil
}

func (c *Client) ReloadConfig(path string, force bool) error {
	body := map[string]string{}
	if path != "" {
		body["path"] = path
	}

	endpoint := "/configs"
	if force {
		endpoint += "?force=true"
	}

	return c.do(http.MethodPut, endpoint, body, nil)
}

func (c *Client) do(method, endpoint string, body any, out any) error {
	requestURL := c.resolve(endpoint)

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, requestURL, reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request mihomo controller: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("mihomo controller returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) resolve(endpoint string) string {
	ref, err := url.Parse(endpoint)
	if err != nil {
		return c.baseURL.String()
	}
	return c.baseURL.ResolveReference(ref).String()
}

func latestDelay(history []delayHistory) int {
	if len(history) == 0 {
		return -1
	}
	return history[len(history)-1].Delay
}
