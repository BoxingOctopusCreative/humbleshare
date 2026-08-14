package steam

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var reStoreAppPath = regexp.MustCompile(`(?i)^(?:/agecheck)?/app/(\d+)(?:/|$)`)

// ParseStoreURL extracts a Steam appid from a store.steampowered.com/app/… URL.
func ParseStoreURL(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("steam store URL is required")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid Steam store URL")
	}
	host := strings.ToLower(u.Hostname())
	if host != "store.steampowered.com" {
		return 0, fmt.Errorf("URL must be a store.steampowered.com app page")
	}
	m := reStoreAppPath.FindStringSubmatch(u.EscapedPath())
	if len(m) != 2 {
		return 0, fmt.Errorf("URL must be a Steam store app page")
	}
	appID, err := strconv.Atoi(m[1])
	if err != nil || appID <= 0 {
		return 0, fmt.Errorf("invalid Steam app id in URL")
	}
	return appID, nil
}
