package humble

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/google/uuid"
)

const apiBase = "https://www.humblebundle.com/api/v1/order"

// steamKeyLike matches typical Steam CD-key shapes (not Humble order gamekeys).
var steamKeyLike = regexp.MustCompile(`(?i)^[A-Z0-9]{4,5}(-[A-Z0-9]{4,5}){2,4}$`)

var (
	reGiftHumanName = regexp.MustCompile(`"human_name"\s*:\s*"((?:\\.|[^"\\])*)"`)
	reGiftOGTitle   = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']+)["']`)
	reGiftTitleTag  = regexp.MustCompile(`(?i)<title[^>]*>\s*([^<]+?)\s*</title>`)
)

// ImportResult reports what happened for a single order key or gift URL.
type ImportResult struct {
	OrderID           string   `json:"order_id"`
	Imported          int      `json:"imported"`
	SkippedDuplicate  int      `json:"skipped_duplicate"`
	SkippedNonSteam   int      `json:"skipped_non_steam"`
	SkippedUnrevealed int      `json:"skipped_unrevealed"`
	Failed            int      `json:"failed"`
	NeedsTitle        bool     `json:"needs_title,omitempty"`
	Messages          []string `json:"messages,omitempty"`
}

// humbleOrder is the minimal subset of the Humble order JSON we care about.
type humbleOrder struct {
	TpkdDict struct {
		AllTpks []humbleTpk `json:"all_tpks"`
	} `json:"tpkd_dict"`
}

type humbleTpk struct {
	HumanName        string `json:"human_name"`
	RedeemedKeyVal   string `json:"redeemed_key_val"`
	KeyTypeHumanName string `json:"key_type_human_name"`
	MachineName      string `json:"machine_name"`
}

func (t *humbleTpk) isSteam() bool {
	return strings.Contains(strings.ToLower(t.KeyTypeHumanName), "steam") ||
		strings.Contains(strings.ToLower(t.MachineName), "steam")
}

// GameEnricher resolves Steam AppID + store metadata from a game title.
type GameEnricher interface {
	EnrichGame(ctx context.Context, gameID uuid.UUID, name string) error
}

// Service fetches Humble order data and imports codes.
// The session cookie is passed in-memory per-call and is never logged.
type Service struct {
	httpClient *http.Client
	codes      repository.CodeRepository
	games      repository.GameRepository
	steam      GameEnricher
}

func NewService(codes repository.CodeRepository, games repository.GameRepository, steam GameEnricher) *Service {
	return &Service{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		codes:      codes,
		games:      games,
		steam:      steam,
	}
}

func (s *Service) enrichGame(ctx context.Context, game *models.Game) string {
	if s.steam == nil || game == nil || game.SteamAppID != nil {
		return ""
	}
	if err := s.steam.EnrichGame(ctx, game.ID, game.Name); err != nil {
		return fmt.Sprintf("steam enrich %q: %v", game.Name, err)
	}
	return ""
}

type pasteKind int

const (
	pasteOrderGamekey pasteKind = iota
	pasteGiftURL
)

// classifyPaste decides whether a line is an order gamekey or a Humble gift URL.
func classifyPaste(raw string) (kind pasteKind, value string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, "", fmt.Errorf("empty order id")
	}
	if steamKeyLike.MatchString(raw) {
		return 0, "", fmt.Errorf(
			"%q looks like a Steam key; paste Steam keys in Bulk paste",
			raw,
		)
	}

	if strings.Contains(raw, "://") || strings.HasPrefix(raw, "//") {
		u, err := url.Parse(raw)
		if err != nil {
			return 0, "", fmt.Errorf("invalid URL: %w", err)
		}
		host := strings.ToLower(u.Hostname())
		if !strings.HasSuffix(host, "humblebundle.com") {
			return 0, "", fmt.Errorf("URL host must be humblebundle.com")
		}
		key := strings.TrimSpace(u.Query().Get("key"))
		if key == "" {
			return 0, "", fmt.Errorf("could not find key= in URL")
		}
		path := strings.ToLower(strings.Trim(u.Path, "/"))
		if path == "gift" || strings.HasPrefix(path, "gift/") {
			// Canonical gift URL used as the shareable code value.
			giftURL := "https://www.humblebundle.com/gift?key=" + url.QueryEscape(key)
			return pasteGiftURL, giftURL, nil
		}
		// downloads?key=… and similar are order gamekeys.
		return pasteOrderGamekey, key, nil
	}

	if strings.ContainsAny(raw, "/?#&= \t") {
		return 0, "", fmt.Errorf("invalid order gamekey %q", raw)
	}
	return pasteOrderGamekey, raw, nil
}

func isHumbleGiftURL(val string) bool {
	u, err := url.Parse(strings.TrimSpace(val))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if !strings.HasSuffix(host, "humblebundle.com") {
		return false
	}
	path := strings.ToLower(strings.Trim(u.Path, "/"))
	return (path == "gift" || strings.HasPrefix(path, "gift/")) && u.Query().Get("key") != ""
}

func codeTypeForValue(val string) string {
	if isHumbleGiftURL(val) {
		return models.CodeTypeGiftLink
	}
	return models.CodeTypeSteamKey
}

// normalizeSessionCookie accepts the raw cookie value or "_simpleauth_sess=…".
func normalizeSessionCookie(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "_simpleauth_sess=") {
		parts := strings.SplitN(raw, "_simpleauth_sess=", 2)
		raw = parts[len(parts)-1]
		if i := strings.Index(raw, ";"); i >= 0 {
			raw = raw[:i]
		}
		raw = strings.TrimSpace(raw)
	}
	if u, err := url.QueryUnescape(raw); err == nil && u != "" {
		raw = u
	}
	return raw
}

// ImportOrders fetches each order key from Humble and upserts game + code rows.
// Gift URLs (https://www.humblebundle.com/gift?key=…) are imported as gift_link codes.
// titles is an optional map of paste line or canonical gift URL → game title, used when
// the gift page does not expose a title (or to skip scraping).
// The sessionCookie value is used only for the HTTP request and is never stored or logged.
func (s *Service) ImportOrders(ctx context.Context, sessionCookie string, orderIDs []string, titles map[string]string) ([]*ImportResult, error) {
	sessionCookie = normalizeSessionCookie(sessionCookie)
	if sessionCookie == "" {
		return nil, fmt.Errorf("session_cookie is required")
	}

	results := make([]*ImportResult, 0, len(orderIDs))
	for _, raw := range orderIDs {
		kind, value, err := classifyPaste(raw)
		if err != nil {
			results = append(results, &ImportResult{
				OrderID:  strings.TrimSpace(raw),
				Failed:   1,
				Messages: []string{err.Error()},
			})
			continue
		}
		var r *ImportResult
		switch kind {
		case pasteGiftURL:
			r, err = s.importGiftURL(ctx, sessionCookie, value, lookupTitle(titles, raw, value))
		default:
			r, err = s.importOrder(ctx, sessionCookie, value)
		}
		if err != nil {
			results = append(results, &ImportResult{
				OrderID:  value,
				Failed:   1,
				Messages: []string{err.Error()},
			})
			continue
		}
		results = append(results, r)
	}
	return results, nil
}

func lookupTitle(titles map[string]string, keys ...string) string {
	if len(titles) == 0 {
		return ""
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if t := strings.TrimSpace(titles[k]); t != "" {
			return t
		}
	}
	return ""
}

func (s *Service) importGiftURL(ctx context.Context, sessionCookie, giftURL, titleOverride string) (*ImportResult, error) {
	result := &ImportResult{OrderID: giftURL}
	title := strings.TrimSpace(titleOverride)
	if title == "" {
		var err error
		title, err = s.resolveGiftTitle(ctx, sessionCookie, giftURL)
		if err != nil {
			result.Failed = 1
			result.NeedsTitle = true
			result.Messages = append(result.Messages, "could not read game title from gift page; enter a title to continue")
			return result, nil
		}
	}
	game, err := s.games.UpsertUnresolvedByName(ctx, title)
	if err != nil {
		result.Failed++
		result.Messages = append(result.Messages, fmt.Sprintf("%s: upsert game: %v", title, err))
		return result, nil
	}
	if note := s.enrichGame(ctx, game); note != "" {
		result.Messages = append(result.Messages, note)
	}
	src := "humble"
	u, _ := url.Parse(giftURL)
	oid := u.Query().Get("key")
	_, err = s.codes.Create(ctx, game.ID, giftURL, models.CodeTypeGiftLink, &src, &oid)
	if errors.Is(err, repository.ErrCodeDuplicate) {
		result.SkippedDuplicate++
		result.Messages = append(result.Messages, fmt.Sprintf("%s: duplicate gift link", title))
		return result, nil
	}
	if err != nil {
		result.Failed++
		result.Messages = append(result.Messages, fmt.Sprintf("%s: insert code: %v", title, err))
		return result, nil
	}
	result.Imported++
	result.Messages = append(result.Messages, fmt.Sprintf("%s: imported gift link", title))
	return result, nil
}

func (s *Service) resolveGiftTitle(ctx context.Context, sessionCookie, giftURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, giftURL, nil)
	if err != nil {
		return "", fmt.Errorf("build gift request: %w", err)
	}
	req.Header.Set("Cookie", "_simpleauth_sess="+sessionCookie)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "HumbleShare/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch gift page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("humble gift page returned %d (check cookie and gift URL)", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("read gift page: %w", err)
	}
	html := string(body)

	if m := reGiftHumanName.FindStringSubmatch(html); len(m) == 2 {
		if name := unescapeJSONString(m[1]); name != "" && !isGenericGiftTitle(name) {
			return name, nil
		}
	}
	if m := reGiftOGTitle.FindStringSubmatch(html); len(m) == 2 {
		if name := strings.TrimSpace(m[1]); name != "" && !isGenericGiftTitle(name) {
			return cleanGiftTitle(name), nil
		}
	}
	if m := reGiftTitleTag.FindStringSubmatch(html); len(m) == 2 {
		if name := cleanGiftTitle(m[1]); name != "" && !isGenericGiftTitle(name) {
			return name, nil
		}
	}
	return "", fmt.Errorf("could not read game title from gift page")
}

func isGenericGiftTitle(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return n == "" ||
		n == "humble bundle" ||
		strings.Contains(n, "your humble bundle receipt") ||
		strings.Contains(n, "a very special gift for you") ||
		strings.Contains(n, "wrong email")
}

func cleanGiftTitle(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, " | Humble Bundle")
	name = strings.TrimSuffix(name, " - Humble Bundle")
	return strings.TrimSpace(name)
}

func unescapeJSONString(s string) string {
	var out string
	if err := json.Unmarshal([]byte(`"`+s+`"`), &out); err != nil {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(out)
}

func (s *Service) importOrder(ctx context.Context, sessionCookie, orderID string) (*ImportResult, error) {
	// all_tpkds=true is required to include redeemed_key_val for Steam keys.
	u := apiBase + "/" + url.PathEscape(orderID) + "?all_tpkds=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Cookie", "_simpleauth_sess="+sessionCookie)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "HumbleShare/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf(
				"humble returned %d for gamekey %s (use downloads?key=… for orders, or gift?key=… gift links; Steam keys go in Bulk paste)",
				resp.StatusCode, orderID,
			)
		}
		return nil, fmt.Errorf("humble API returned %d for order %s", resp.StatusCode, orderID)
	}

	var order humbleOrder
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, fmt.Errorf("decode order: %w", err)
	}

	result := &ImportResult{OrderID: orderID}
	source := "humble"

	for _, tpk := range order.TpkdDict.AllTpks {
		if strings.TrimSpace(tpk.RedeemedKeyVal) == "" {
			result.SkippedUnrevealed++
			continue
		}
		codeValue := strings.TrimSpace(tpk.RedeemedKeyVal)
		codeType := codeTypeForValue(codeValue)
		// Gift links are Steam entitlement gifts; non-Steam non-gift keys are skipped.
		if codeType != models.CodeTypeGiftLink && !tpk.isSteam() {
			result.SkippedNonSteam++
			continue
		}

		name := strings.TrimSpace(tpk.HumanName)
		if name == "" {
			result.SkippedNonSteam++
			continue
		}

		game, err := s.games.UpsertUnresolvedByName(ctx, name)
		if err != nil {
			result.Failed++
			result.Messages = append(result.Messages, fmt.Sprintf("%s: upsert game: %v", name, err))
			continue
		}
		if note := s.enrichGame(ctx, game); note != "" {
			result.Messages = append(result.Messages, note)
		}

		srcStr := source
		oidStr := orderID
		_, err = s.codes.Create(ctx, game.ID, codeValue, codeType, &srcStr, &oidStr)
		if errors.Is(err, repository.ErrCodeDuplicate) {
			result.SkippedDuplicate++
			result.Messages = append(result.Messages, fmt.Sprintf("%s: duplicate", name))
			continue
		}
		if err != nil {
			result.Failed++
			result.Messages = append(result.Messages, fmt.Sprintf("%s: insert code: %v", name, err))
			continue
		}
		result.Imported++
		kind := "steam key"
		if codeType == models.CodeTypeGiftLink {
			kind = "gift link"
		}
		result.Messages = append(result.Messages, fmt.Sprintf("%s: imported %s", name, kind))
	}

	return result, nil
}
