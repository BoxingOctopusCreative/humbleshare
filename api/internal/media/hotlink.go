package media

import (
	"net/url"
	"strings"
)

// AllowedHosts builds a host allowlist from absolute URLs (web base, CORS origins).
func AllowedHosts(rawURLs ...string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, raw := range rawURLs {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if !strings.Contains(part, "://") {
				part = "https://" + part
			}
			u, err := url.Parse(part)
			if err != nil || u.Host == "" {
				continue
			}
			out[strings.ToLower(u.Host)] = struct{}{}
		}
	}
	return out
}

// RefererAllowed implements Cloudflare-style hotlink checks:
// blank Referer/Origin is allowed; a present foreign host is denied.
func RefererAllowed(referer, origin string, hosts map[string]struct{}) bool {
	if len(hosts) == 0 {
		return true
	}
	for _, raw := range []string{referer, origin} {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return false
		}
		if _, ok := hosts[strings.ToLower(u.Host)]; ok {
			return true
		}
		return false
	}
	return true
}

// ParseObjectKey validates media object keys (avatars|banners)/filename.
func ParseObjectKey(kind, name string) (string, bool) {
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	if kind != "avatars" && kind != "banners" {
		return "", false
	}
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") || strings.Contains(name, "\\") {
		return "", false
	}
	if strings.HasPrefix(name, ".") {
		return "", false
	}
	return kind + "/" + name, true
}

// ProxyURL rewrites a CDN asset URL to a same-origin /media/... path on webBase.
func ProxyURL(webBase, cdnBase, assetURL string) string {
	webBase = strings.TrimRight(strings.TrimSpace(webBase), "/")
	cdnBase = strings.TrimRight(strings.TrimSpace(cdnBase), "/")
	assetURL = strings.TrimSpace(assetURL)
	if webBase == "" || cdnBase == "" || assetURL == "" {
		return assetURL
	}
	prefix := cdnBase + "/"
	if !strings.HasPrefix(assetURL, prefix) {
		return assetURL
	}
	key := strings.TrimPrefix(assetURL, prefix)
	if key == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return assetURL
	}
	if !strings.HasPrefix(key, "avatars/") && !strings.HasPrefix(key, "banners/") {
		return assetURL
	}
	return webBase + "/media/" + key
}
