package steam

import "strings"

// MatchExactAppID returns the Steam appid for an exact case-insensitive name match,
// or 0 when nothing matches closely enough to trust.
func MatchExactAppID(title string, candidates []SearchResult) int {
	nameLower := strings.ToLower(strings.TrimSpace(title))
	if nameLower == "" {
		return 0
	}
	for _, r := range candidates {
		if strings.ToLower(strings.TrimSpace(r.Name)) == nameLower {
			return r.AppID
		}
	}
	return 0
}
