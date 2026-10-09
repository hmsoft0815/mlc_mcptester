package authcheck

import (
	"context"
	"net/http"
	"slices"
	"strings"
)

// Check runs discovery and all checks against endpoint.
func Check(ctx context.Context, endpoint string, c *http.Client) *Report {
	rep := &Report{Endpoint: endpoint}
	d, err := Discover(ctx, endpoint, c)
	if d == nil && err == nil {
		rep.add("authorization", Info, "the endpoint answers without authorization (no 401)")
		return rep
	}
	rep.Protected = true
	rep.Discovery = d
	if d == nil {
		rep.add("unauthorized request", Fail, "%v", err)
		return rep
	}
	rep.ResourceMeta, rep.AuthServer = d.resourceMeta, d.asMeta

	checkChallenge(rep, d.Challenge)
	if err != nil {
		rep.add("Protected Resource Metadata", Fail, "%v", err)
		return rep
	}
	rep.add("Protected Resource Metadata", Pass, "%s", d.MetadataURL)
	if !checkResourceMetadata(rep, d, endpoint) {
		return rep
	}
	checkIssuer(rep, d)
	checkFlows(rep, d.asMeta)
	return rep
}

// checkChallenge rates the 401 challenge (RFC 9728 §5.1).
func checkChallenge(rep *Report, challenge string) {
	switch {
	case !strings.HasPrefix(strings.ToLower(challenge), "bearer"):
		rep.add("WWW-Authenticate", Warn, "401 without a Bearer challenge (%q); clients fall back to the well-known URI", challenge)
	case resourceMetadataParam.MatchString(challenge):
		rep.add("WWW-Authenticate", Pass, "Bearer with resource_metadata")
	default:
		rep.add("WWW-Authenticate", Info, "Bearer without resource_metadata; clients use the well-known URI")
	}
}

// checkResourceMetadata checks resource and authorization_servers (RFC 9728,
// MCP authorization) and that the authorization server publishes metadata.
// It reports whether the checks can go on.
func checkResourceMetadata(rep *Report, d *Discovery, endpoint string) bool {
	switch {
	case d.Resource == "":
		rep.add("resource", Fail, "Protected Resource Metadata has no resource")
	case canonical(d.Resource) != canonical(endpoint) && !strings.HasPrefix(canonical(endpoint), canonical(d.Resource)):
		rep.add("resource", Warn, "resource %q does not identify the endpoint %q; tokens are audience-bound to it (RFC 8707)", d.Resource, endpoint)
	default:
		rep.add("resource", Pass, "%s", d.Resource)
	}
	if len(d.AuthServers) == 0 {
		rep.add("authorization_servers", Fail, "no authorization server advertised")
		return false
	}
	rep.add("authorization_servers", Pass, "%s", strings.Join(d.AuthServers, ", "))
	if d.asMeta == nil {
		rep.add("authorization server metadata", Fail, "none found for %s (RFC 8414 / OIDC discovery)", d.AuthServer())
		return false
	}
	rep.add("authorization server metadata", Pass, "%s", d.ASMetadataURL)
	return true
}

// checkIssuer: the metadata names the advertised server (RFC 8414 §3.3) and
// a token endpoint.
func checkIssuer(rep *Report, d *Discovery) {
	if iss, _ := d.asMeta["issuer"].(string); strings.TrimSuffix(iss, "/") != strings.TrimSuffix(d.AuthServer(), "/") {
		rep.add("issuer", Fail, "metadata issuer %q differs from the advertised authorization server %q (RFC 8414 §3.3)", iss, d.AuthServer())
	} else {
		rep.add("issuer", Pass, "%s", iss)
	}
	if s, _ := d.asMeta["token_endpoint"].(string); s == "" {
		rep.add("token_endpoint", Fail, "missing")
	}
}

// asCapabilities are the grants and token endpoint auth methods, with the
// RFC 8414 defaults where the metadata names none.
type asCapabilities struct {
	grants, authMethods []string
}

func capabilitiesOf(m map[string]any) asCapabilities {
	caps := asCapabilities{
		grants:      stringList(m["grant_types_supported"]),
		authMethods: stringList(m["token_endpoint_auth_methods_supported"]),
	}
	if len(caps.grants) == 0 {
		caps.grants = []string{"authorization_code", "implicit"} // RFC 8414 default
	}
	if len(caps.authMethods) == 0 {
		caps.authMethods = []string{"client_secret_basic"} // RFC 8414 default
	}
	return caps
}

// checkFlows lists the flows the authorization server offers and checks each.
func checkFlows(rep *Report, m map[string]any) {
	caps := capabilitiesOf(m)
	checkAuthCode(rep, m, caps)
	checkClientCredentials(rep, m, caps)
	checkEnterprise(rep, m, caps)
}

// checkAuthCode: authorization code with PKCE, the default flow of the core spec.
func checkAuthCode(rep *Report, m map[string]any, caps asCapabilities) {
	authCode := slices.Contains(caps.grants, "authorization_code") && m["authorization_endpoint"] != nil
	if authCode {
		if slices.Contains(stringList(m["code_challenge_methods_supported"]), "S256") {
			rep.add("PKCE (S256)", Pass, "code_challenge_methods_supported includes S256")
		} else {
			rep.add("PKCE (S256)", Fail, "code_challenge_methods_supported lacks S256; MCP clients must refuse to proceed")
		}
		if b, _ := m["authorization_response_iss_parameter_supported"].(bool); b {
			rep.add("iss in authorization response", Pass, "RFC 9207 supported")
		} else {
			rep.add("iss in authorization response", Warn, "authorization_response_iss_parameter_supported is not true; SHOULD per spec 2026-07-28 (RFC 9207)")
		}
	}
	registration := []string{}
	if b, _ := m["client_id_metadata_document_supported"].(bool); b {
		registration = append(registration, "Client ID Metadata Documents")
	}
	if m["registration_endpoint"] != nil {
		registration = append(registration, "dynamic registration (deprecated since 2026-07-28)")
	}
	rep.Flows = append(rep.Flows, Flow{
		Name: "Authorization code + PKCE", Supported: authCode, Flag: "--oauth",
		Detail: "registration: " + orNone(registration) + "; pre-registered clients: --oauth-client-id",
	})
}

// checkClientCredentials: OAuth client credentials (ext-auth).
func checkClientCredentials(rep *Report, m map[string]any, caps asCapabilities) {
	cc := slices.Contains(caps.grants, "client_credentials")
	if cc {
		if !slices.Contains(caps.authMethods, "private_key_jwt") && !slices.Contains(caps.authMethods, "client_secret_basic") {
			rep.add("client credentials: auth methods", Fail, "token_endpoint_auth_methods_supported must include private_key_jwt or client_secret_basic (ext-auth)")
		} else {
			rep.add("client credentials: auth methods", Pass, "%s", strings.Join(caps.authMethods, ", "))
		}
		if slices.Contains(caps.authMethods, "private_key_jwt") && m["token_endpoint_auth_signing_alg_values_supported"] == nil {
			rep.add("client credentials: signing algorithms", Fail, "private_key_jwt without token_endpoint_auth_signing_alg_values_supported (ext-auth)")
		}
	}
	rep.Flows = append(rep.Flows, Flow{
		Name: "Client credentials (ext-auth)", Supported: cc, Flag: "--oauth-client-credentials",
		Detail: "token endpoint auth: " + strings.Join(caps.authMethods, ", "),
	})
}

// checkEnterprise: enterprise-managed authorization with ID-JAG (ext-auth).
func checkEnterprise(rep *Report, m map[string]any, caps asCapabilities) {
	ema := slices.Contains(stringList(m["authorization_grant_profiles_supported"]), "urn:ietf:params:oauth:grant-profile:id-jag")
	if ema && !slices.Contains(caps.grants, "urn:ietf:params:oauth:grant-type:jwt-bearer") && m["grant_types_supported"] != nil {
		rep.add("enterprise: jwt-bearer grant", Warn, "the id-jag profile is advertised, but grant_types_supported lacks urn:ietf:params:oauth:grant-type:jwt-bearer")
	}
	rep.Flows = append(rep.Flows, Flow{Name: "Enterprise-managed (ID-JAG, ext-auth)", Supported: ema, Flag: "--oauth-enterprise"})
}
