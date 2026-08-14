package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/google/uuid"
)

const (
	storeSearchURL = "https://store.steampowered.com/api/storesearch/?term=%s&cc=CA&l=en"
	appDetailsURL  = "https://store.steampowered.com/api/appdetails?appids=%d&cc=CA&l=en"
	maxSuggestions = 8
)

// AppMetadata holds Steam metadata fetched from appdetails.
type AppMetadata struct {
	AppID            int
	Name             string
	ShortDescription string
	HeaderImageURL   string
	CapsuleImageURL  string
	StoreURL         string
	ScreenshotURLs   []string
}

// metaCacheEntry wraps metadata with a fetch timestamp.
type metaCacheEntry struct {
	meta      *AppMetadata
	fetchedAt time.Time
}

// tokenBucket is a simple in-memory rate limiter.
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	maxRate  float64 // per second
	lastTime time.Time
}

func newTokenBucket(ratePerSecond float64) *tokenBucket {
	return &tokenBucket{
		tokens:   ratePerSecond,
		maxRate:  ratePerSecond,
		lastTime: time.Now(),
	}
}

func (b *tokenBucket) wait(ctx context.Context) error {
	for {
		b.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(b.lastTime).Seconds()
		b.tokens = min(b.tokens+elapsed*b.maxRate, b.maxRate)
		b.lastTime = now
		if b.tokens >= 1.0 {
			b.tokens -= 1.0
			b.mu.Unlock()
			return nil
		}
		waitTime := time.Duration((1.0-b.tokens)/b.maxRate*float64(time.Second)) + time.Millisecond
		b.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}
}

// Client fetches Steam store data with rate limiting and an in-memory metadata cache.
type Client struct {
	httpClient *http.Client
	limiter    *tokenBucket
	cacheMu    sync.RWMutex
	cache      map[int]*metaCacheEntry
	cacheTTL   time.Duration
	gameRepo   repository.GameRepository
}

func NewClient(ratePerSecond float64, cacheTTL time.Duration, gameRepo repository.GameRepository) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		limiter:    newTokenBucket(ratePerSecond),
		cache:      make(map[int]*metaCacheEntry),
		cacheTTL:   cacheTTL,
		gameRepo:   gameRepo,
	}
}

// StoreSearch returns candidate apps matching title.
type SearchResult struct {
	AppID int    `json:"appid"`
	Name  string `json:"name"`
}

// StoreSearch queries the Steam store search API and returns candidates.
func (c *Client) StoreSearch(ctx context.Context, title string) ([]SearchResult, error) {
	if err := c.limiter.wait(ctx); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(storeSearchURL, url.QueryEscape(title))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("steam storesearch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam storesearch: status %d", resp.StatusCode)
	}

	var payload struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("steam storesearch: decode: %w", err)
	}

	results := make([]SearchResult, 0, len(payload.Items))
	for _, it := range payload.Items {
		results = append(results, SearchResult{AppID: it.ID, Name: it.Name})
	}
	return results, nil
}

// AppDetails fetches metadata for a Steam appid. Results are cached by appid with TTL.
func (c *Client) AppDetails(ctx context.Context, appID int) (*AppMetadata, error) {
	// Check cache.
	c.cacheMu.RLock()
	if entry, ok := c.cache[appID]; ok && time.Since(entry.fetchedAt) < c.cacheTTL {
		c.cacheMu.RUnlock()
		return entry.meta, nil
	}
	c.cacheMu.RUnlock()

	if err := c.limiter.wait(ctx); err != nil {
		return nil, err
	}

	u := fmt.Sprintf(appDetailsURL, appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("steam appdetails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam appdetails: status %d", resp.StatusCode)
	}

	// The appdetails API returns a map keyed by appid.
	var raw map[string]struct {
		Success bool `json:"success"`
		Data    struct {
			Name             string `json:"name"`
			ShortDescription string `json:"short_description"`
			HeaderImage      string `json:"header_image"`
			CapsuleImage     string `json:"capsule_image"`
			Screenshots      []struct {
				PathFull      string `json:"path_full"`
				PathThumbnail string `json:"path_thumbnail"`
			} `json:"screenshots"`
			Movies []struct {
				Thumbnail string `json:"thumbnail"`
			} `json:"movies"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("steam appdetails: decode: %w", err)
	}

	key := fmt.Sprintf("%d", appID)
	entry, ok := raw[key]
	if !ok || !entry.Success {
		return nil, fmt.Errorf("steam appdetails: no data for appid %d", appID)
	}

	storeURL := fmt.Sprintf("https://store.steampowered.com/app/%d", appID)
	screenshots := make([]string, 0, len(entry.Data.Screenshots)+len(entry.Data.Movies))
	for _, shot := range entry.Data.Screenshots {
		url := shot.PathFull
		if url == "" {
			url = shot.PathThumbnail
		}
		if url != "" {
			screenshots = append(screenshots, url)
		}
	}
	// Trailer / footage thumbnails give the carousel in-game motion stills.
	for _, movie := range entry.Data.Movies {
		if movie.Thumbnail != "" {
			screenshots = append(screenshots, movie.Thumbnail)
		}
	}

	meta := &AppMetadata{
		AppID:            appID,
		Name:             entry.Data.Name,
		ShortDescription: entry.Data.ShortDescription,
		HeaderImageURL:   entry.Data.HeaderImage,
		CapsuleImageURL:  entry.Data.CapsuleImage,
		StoreURL:         storeURL,
		ScreenshotURLs:   screenshots,
	}

	c.cacheMu.Lock()
	c.cache[appID] = &metaCacheEntry{meta: meta, fetchedAt: time.Now()}
	c.cacheMu.Unlock()

	return meta, nil
}

// EnrichGame searches Steam for the game by name and updates its metadata in the DB.
// Returns nil if no exact match is found (leaves game unresolved rather than failing).
func (c *Client) EnrichGame(ctx context.Context, gameID uuid.UUID, name string) error {
	_, _, err := c.ResolveByName(ctx, gameID, name)
	return err
}

// ResolveByName looks up a game by exact title. On an exact match it sets steam_appid
// and refreshes store metadata. Otherwise it returns similar store-search titles.
func (c *Client) ResolveByName(ctx context.Context, gameID uuid.UUID, name string) (appID int, suggestions []SearchResult, err error) {
	candidates, err := c.StoreSearch(ctx, name)
	if err != nil {
		return 0, nil, fmt.Errorf("search %q: %w", name, err)
	}

	matchedAppID := MatchExactAppID(name, candidates)
	if matchedAppID == 0 {
		return 0, limitSuggestions(candidates, maxSuggestions), nil
	}

	if err := c.gameRepo.SetSteamAppID(ctx, gameID, matchedAppID); err != nil {
		return 0, nil, fmt.Errorf("set steam_appid: %w", err)
	}
	if err := c.RefreshMetadata(ctx, gameID, matchedAppID); err != nil {
		return 0, nil, err
	}
	return matchedAppID, nil, nil
}

func limitSuggestions(in []SearchResult, n int) []SearchResult {
	if len(in) <= n {
		return in
	}
	out := make([]SearchResult, n)
	copy(out, in)
	return out
}

// RefreshMetadata fetches the latest Steam metadata for appID and writes it to the DB.
func (c *Client) RefreshMetadata(ctx context.Context, gameID uuid.UUID, appID int) error {
	c.cacheMu.Lock()
	delete(c.cache, appID)
	c.cacheMu.Unlock()

	meta, err := c.AppDetails(ctx, appID)
	if err != nil {
		return err
	}

	patch := repository.GameMetadataPatch{
		ShortDescription: strPtr(meta.ShortDescription),
		StoreURL:         strPtr(meta.StoreURL),
		ScreenshotURLs:   &meta.ScreenshotURLs,
	}
	if meta.HeaderImageURL != "" {
		patch.HeaderImageURL = strPtr(meta.HeaderImageURL)
	}
	if meta.CapsuleImageURL != "" {
		patch.CapsuleImageURL = strPtr(meta.CapsuleImageURL)
	}

	return c.gameRepo.UpdateMetadata(ctx, gameID, patch)
}

// StartEnrichmentWorker launches a background goroutine that periodically attempts to resolve
// unresolved games (those without a steam_appid) against the Steam store.
func (c *Client) StartEnrichmentWorker(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.runEnrichmentCycle(ctx)
			}
		}
	}()
}

func (c *Client) runEnrichmentCycle(ctx context.Context) {
	games, err := c.gameRepo.ListUnresolved(ctx)
	if err != nil {
		log.Printf("steam enrichment: list unresolved: %v", err)
		return
	}
	for _, g := range games {
		if ctx.Err() != nil {
			return
		}
		if err := c.EnrichGame(ctx, g.ID, g.Name); err != nil {
			log.Printf("steam enrichment: %s: %v", g.Name, err)
		}
	}

	missing, err := c.gameRepo.ListMissingScreenshots(ctx)
	if err != nil {
		log.Printf("steam enrichment: list missing screenshots: %v", err)
		return
	}
	for _, g := range missing {
		if ctx.Err() != nil {
			return
		}
		if g.SteamAppID == nil {
			continue
		}
		if err := c.RefreshMetadata(ctx, g.ID, *g.SteamAppID); err != nil {
			log.Printf("steam enrichment: screenshots %s: %v", g.Name, err)
		}
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
