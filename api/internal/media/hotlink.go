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

// ParseObjectKey validates media object keys (avatars|banners|svg)/filename.
func ParseObjectKey(kind, name string) (string, bool) {
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") || strings.Contains(name, "\\") {
		return "", false
	}
	if strings.HasPrefix(name, ".") {
		return "", false
	}
	switch kind {
	case "avatars", "banners":
		return kind + "/" + name, true
	case "svg":
		if !strings.HasSuffix(strings.ToLower(name), ".svg") {
			return "", false
		}
		return "SVG/" + name, true
	default:
		return "", false
	}
}

// ProxyURL rewrites a stored CDN asset URL to a same-origin relative /media/... path.
func ProxyURL(cdnBase, assetURL string) string {
	assetURL = strings.TrimSpace(assetURL)
	if assetURL == "" {
		return assetURL
	}
	if u, err := url.Parse(assetURL); err == nil {
		key := strings.TrimPrefix(u.Path, "/")
		if objectKeyOK(key) {
			return "/media/" + key
		}
	}
	cdnBase = strings.TrimRight(strings.TrimSpace(cdnBase), "/")
	if cdnBase != "" && strings.HasPrefix(assetURL, cdnBase+"/") {
		key := strings.TrimPrefix(assetURL, cdnBase+"/")
		if objectKeyOK(key) {
			return "/media/" + key
		}
	}
	return assetURL
}

func objectKeyOK(key string) bool {
	if key == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return false
	}
	return strings.HasPrefix(key, "avatars/") || strings.HasPrefix(key, "banners/")
}
