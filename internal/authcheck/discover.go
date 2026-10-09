package authcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var resourceMetadataParam = regexp.MustCompile(`resource_metadata="([^"]+)"`)

// Discover finds the Protected Resource Metadata (from the 401 challenge,
// else the well-known URIs) and the first authorization server's metadata.
// It returns nil, nil if the endpoint does not require authorization.
func Discover(ctx context.Context, endpoint string, c *http.Client) (*Discovery, error) {
	if c == nil {
		c = http.DefaultClient
	}
	challenge, protected, err := probe(ctx, endpoint, c)
	if err != nil || !protected {
		return nil, err
	}
	d := &Discovery{Challenge: challenge}
	if err := d.findResourceMetadata(ctx, c, endpoint); err != nil {
		return d, err
	}
	d.findAuthServerMetadata(ctx, c)
	return d, nil
}

// probe sends an unauthorized server/discover and returns the challenge of a 401.
func probe(ctx context.Context, endpoint string, c *http.Client) (challenge string, protected bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{}}`))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := c.Do(req)
	if err != nil {
		return "", false, err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		return "", false, nil
	}
	return resp.Header.Get("WWW-Authenticate"), true, nil
}

// findResourceMetadata reads the Protected Resource Metadata named in the
// challenge, else at the well-known URIs, with resource and authorization servers.
func (d *Discovery) findResourceMetadata(ctx context.Context, c *http.Client, endpoint string) error {
	var candidates []string
	if m := resourceMetadataParam.FindStringSubmatch(d.Challenge); m != nil {
		candidates = append(candidates, m[1])
	}
	candidates = append(candidates, wellKnown(endpoint, "oauth-protected-resource")...)
	for _, u := range candidates {
		if meta, err := getJSON(ctx, c, u); err == nil {
			d.MetadataURL, d.resourceMeta = u, meta
			break
		}
	}
	if d.resourceMeta == nil {
		return fmt.Errorf("no Protected Resource Metadata found (tried %s)", strings.Join(candidates, ", "))
	}
	d.Resource, _ = d.resourceMeta["resource"].(string)
	d.AuthServers = stringList(d.resourceMeta["authorization_servers"])
	return nil
}

// findAuthServerMetadata reads the first authorization server's metadata.
func (d *Discovery) findAuthServerMetadata(ctx context.Context, c *http.Client) {
	issuer := d.AuthServer()
	if issuer == "" {
		return
	}
	for _, u := range authServerMetadataURLs(issuer) {
		if meta, err := getJSON(ctx, c, u); err == nil {
			d.ASMetadataURL, d.asMeta = u, meta
			return
		}
	}
}

// wellKnown returns the path-inserted and root well-known URIs (RFC 8615 / 9728).
func wellKnown(resource, name string) []string {
	u, err := url.Parse(resource)
	if err != nil {
		return nil
	}
	root := u.Scheme + "://" + u.Host + "/.well-known/" + name
	if p := strings.TrimSuffix(u.Path, "/"); p != "" {
		return []string{root + p, root}
	}
	return []string{root}
}

// authServerMetadataURLs lists the RFC 8414 and OIDC discovery URLs in the
// order the spec prescribes.
func authServerMetadataURLs(issuer string) []string {
	u, err := url.Parse(issuer)
	if err != nil {
		return nil
	}
	base := u.Scheme + "://" + u.Host
	p := strings.TrimSuffix(u.Path, "/")
	if p == "" {
		return []string{base + "/.well-known/oauth-authorization-server", base + "/.well-known/openid-configuration"}
	}
	return []string{
		base + "/.well-known/oauth-authorization-server" + p,
		base + "/.well-known/openid-configuration" + p,
		base + p + "/.well-known/openid-configuration",
	}
}

func getJSON(ctx context.Context, c *http.Client, u string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
	}
	var m map[string]any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m); err != nil {
		return nil, fmt.Errorf("%s: %w", u, err)
	}
	return m, nil
}
