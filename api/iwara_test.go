package api

import (
	"bytes"
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/tls-client/profiles"
)

func headerValue(h map[string][]string, key string) string {
	for headerKey, values := range h {
		if strings.EqualFold(headerKey, key) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func TestSwitchHeadersUsesAPIFetchMetadata(t *testing.T) {
	h := SwitchHeaders("www.iwara.tv")

	if got := headerValue(h, "accept"); got != "application/json, text/plain, */*" {
		t.Fatalf("accept = %q, want API fetch accept header", got)
	}
	if got := headerValue(h, "sec-fetch-dest"); got != "empty" {
		t.Fatalf("sec-fetch-dest = %q, want empty", got)
	}
	if got := headerValue(h, "sec-fetch-mode"); got != "cors" {
		t.Fatalf("sec-fetch-mode = %q, want cors", got)
	}
	if got := headerValue(h, "sec-fetch-site"); got != "same-site" {
		t.Fatalf("sec-fetch-site = %q, want same-site", got)
	}
	if got := headerValue(h, "origin"); got != "https://www.iwara.tv" {
		t.Fatalf("origin = %q, want site origin", got)
	}
	if got := headerValue(h, "referer"); got != "https://www.iwara.tv/" {
		t.Fatalf("referer = %q, want site referer", got)
	}
}

func TestSwitchHeadersDropsNavigationOnlyHeaders(t *testing.T) {
	h := SwitchHeaders("www.iwara.tv")

	if got := headerValue(h, "content-type"); got != "" {
		t.Fatalf("content-type = %q, want empty for GET fetches", got)
	}
	if got := headerValue(h, "sec-fetch-user"); got != "" {
		t.Fatalf("sec-fetch-user = %q, want empty for API fetches", got)
	}
	if got := headerValue(h, "upgrade-insecure-requests"); got != "" {
		t.Fatalf("upgrade-insecure-requests = %q, want empty for API fetches", got)
	}
}

func TestSwitchHeadersUsesStructuredClientHints(t *testing.T) {
	h := SwitchHeaders("www.iwara.tv")
	got := headerValue(h, "sec-ch-ua")

	if strings.Contains(got, `\"`) {
		t.Fatalf("sec-ch-ua = %q, want quoted brands without literal backslashes", got)
	}
	if !strings.Contains(got, `"Google Chrome"`) {
		t.Fatalf("sec-ch-ua = %q, want Google Chrome brand token", got)
	}
}

func TestSwitchHeadersUseChromeVersionSupportedByTLSProfile(t *testing.T) {
	h := SwitchHeaders("www.iwara.tv")

	if got := headerValue(h, "user-agent"); !strings.Contains(got, "Chrome/146.") {
		t.Fatalf("user-agent = %q, want Chrome 146 to match available TLS profile", got)
	}
	if got := headerValue(h, "sec-ch-ua"); !strings.Contains(got, `"146"`) {
		t.Fatalf("sec-ch-ua = %q, want Chrome 146 client hint version", got)
	}
}

func TestDefaultClientProfileMatchesHeaderVersion(t *testing.T) {
	if got := defaultClientProfile().GetClientHelloStr(); got != profiles.Chrome_146_PSK.GetClientHelloStr() {
		t.Fatalf("client profile = %q, want %q", got, profiles.Chrome_146_PSK.GetClientHelloStr())
	}
}

func TestFormatHTTPErrorIncludesCloudflareDiagnostics(t *testing.T) {
	resp := &http.Response{
		StatusCode: 403,
		Header: http.Header{
			"cf-mitigated": {"challenge"},
			"server":       {"cloudflare"},
			"content-type": {"text/html; charset=UTF-8"},
		},
	}
	body := []byte("<!DOCTYPE html><html><title>Just a moment...</title><body>Enable JavaScript and cookies to continue</body></html>")

	err := formatHTTPError(resp, body)
	if err == nil {
		t.Fatal("formatHTTPError returned nil")
	}
	msg := err.Error()

	for _, want := range []string{
		"http status code: 403",
		"cf-mitigated=challenge",
		"server=cloudflare",
		"content-type=text/html; charset=UTF-8",
		"body=<!DOCTYPE html><html><title>Just a moment...",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not contain %q", msg, want)
		}
	}
	if strings.Contains(msg, "\n") || strings.Contains(msg, "\t") {
		t.Fatalf("error %q should normalize body whitespace", msg)
	}
	if len(msg) > 280 {
		t.Fatalf("error %q should stay compact", msg)
	}
	if bytes.Count([]byte(msg), []byte("body=")) != 1 {
		t.Fatalf("error %q should contain one body summary", msg)
	}
}
