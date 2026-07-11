package subscription

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxSubscriptionBytes = 8 << 20

type Fetcher struct {
	Client       *http.Client
	AllowPrivate bool
}

func DefaultFetcher() Fetcher {
	fetcher := Fetcher{}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = fetcher.safeDialContext(dialer)
	fetcher.Client = &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many subscription redirects")
			}
			return validateRemoteURL(request.URL, false)
		},
	}
	return fetcher
}

func (f Fetcher) Import(location, sourceID string) (ImportResult, error) {
	parsed, err := url.Parse(strings.TrimSpace(location))
	if err != nil || validateRemoteURL(parsed, f.AllowPrivate) != nil {
		return ImportResult{}, fmt.Errorf("subscription URL must be a valid HTTP or HTTPS URL")
	}
	client := f.Client
	if client == nil {
		client = DefaultFetcher().Client
	}
	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return ImportResult{}, fmt.Errorf("create subscription request: %w", err)
	}
	req.Header.Set("User-Agent", "mihomo-tui")
	req.Header.Set("Accept", "text/yaml, text/plain, application/yaml, */*")
	resp, err := client.Do(req)
	if err != nil {
		return ImportResult{}, fmt.Errorf("download subscription: %w", sanitizeNetworkError(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ImportResult{}, fmt.Errorf("subscription server returned %s", resp.Status)
	}
	if resp.ContentLength > maxSubscriptionBytes {
		return ImportResult{}, fmt.Errorf("subscription response exceeds 8 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSubscriptionBytes+1))
	if err != nil {
		return ImportResult{}, fmt.Errorf("read subscription response: %w", err)
	}
	if len(data) > maxSubscriptionBytes {
		return ImportResult{}, fmt.Errorf("subscription response exceeds 8 MiB")
	}
	return ImportContent(data, sourceID)
}

func (f Fetcher) safeDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid subscription address")
		}
		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve subscription host: %w", err)
		}
		for _, address := range addresses {
			if !f.AllowPrivate && unsafeIP(address.IP) {
				return nil, fmt.Errorf("subscription host resolves to a private address")
			}
		}
		var lastErr error
		for _, resolved := range addresses {
			connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.IP.String(), port))
			if err == nil {
				return connection, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
}

func validateRemoteURL(parsed *url.URL, allowPrivate bool) error {
	if parsed == nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Hostname() == "" || parsed.User != nil {
		return fmt.Errorf("invalid remote URL")
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && !allowPrivate && unsafeIP(ip) {
		return fmt.Errorf("private address is not allowed")
	}
	return nil
}

func unsafeIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

func sanitizeNetworkError(err error) error {
	// HTTP errors often include the full URL, including subscription tokens.
	// Return a stable message instead of exposing the wrapped text.
	if err == nil {
		return nil
	}
	return fmt.Errorf("request failed")
}
