package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	oauthStateTyp = "oauth_state"
	oauthStateTTL = 10 * time.Minute
)

type OAuthService struct {
	auth     *AuthService
	oauth    repository.OAuthIdentityRepository
	users    repository.UserRepository
	jwtSecret []byte
	publicWeb string
	google    *oauth2.Config
	discord   *oauth2.Config
	steamKey  string
}

func NewOAuthService(
	auth *AuthService,
	oauth repository.OAuthIdentityRepository,
	users repository.UserRepository,
	jwtSecret, publicWebBaseURL string,
	googleID, googleSecret, discordID, discordSecret, steamAPIKey string,
) *OAuthService {
	base := strings.TrimRight(strings.TrimSpace(publicWebBaseURL), "/")
	s := &OAuthService{
		auth:      auth,
		oauth:     oauth,
		users:     users,
		jwtSecret: []byte(jwtSecret),
		publicWeb: base,
		steamKey:  strings.TrimSpace(steamAPIKey),
	}
	if googleID != "" && googleSecret != "" {
		s.google = &oauth2.Config{
			ClientID:     googleID,
			ClientSecret: googleSecret,
			RedirectURL:  base + "/api/v1/auth/oauth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}
	}
	if discordID != "" && discordSecret != "" {
		s.discord = &oauth2.Config{
			ClientID:     discordID,
			ClientSecret: discordSecret,
			RedirectURL:  base + "/api/v1/auth/oauth/discord/callback",
			Scopes:       []string{"identify", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/api/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
		}
	}
	return s
}

func (s *OAuthService) AuthURL(provider, mode, userID string) (string, error) {
	state, err := s.signState(provider, mode, userID)
	if err != nil {
		return "", err
	}
	switch provider {
	case "google":
		if s.google == nil {
			return "", fmt.Errorf("google oauth not configured")
		}
		return s.google.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
	case "discord":
		if s.discord == nil {
			return "", fmt.Errorf("discord oauth not configured")
		}
		return s.discord.AuthCodeURL(state), nil
	case "steam":
		if s.steamKey == "" {
			return "", fmt.Errorf("steam oauth not configured")
		}
		return s.steamAuthURL(state), nil
	default:
		return "", fmt.Errorf("unknown provider")
	}
}

func (s *OAuthService) steamAuthURL(state string) string {
	realm := s.publicWeb
	returnURL := s.publicWeb + "/api/v1/auth/oauth/steam/callback?state=" + url.QueryEscape(state)
	q := url.Values{}
	q.Set("openid.ns", "http://specs.openid.net/auth/2.0")
	q.Set("openid.mode", "checkid_setup")
	q.Set("openid.return_to", returnURL)
	q.Set("openid.realm", realm)
	q.Set("openid.identity", "http://specs.openid.net/auth/2.0/identifier_select")
	q.Set("openid.claimed_id", "http://specs.openid.net/auth/2.0/identifier_select")
	return "https://steamcommunity.com/openid/login?" + q.Encode()
}

func (s *OAuthService) signState(provider, mode, userID string) (string, error) {
	claims := jwt.MapClaims{
		"typ": oauthStateTyp,
		"p":   provider,
		"m":   mode,
		"exp": time.Now().Add(oauthStateTTL).Unix(),
	}
	if mode == "link" && userID != "" {
		claims["uid"] = userID
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *OAuthService) parseState(token string) (provider, mode, userID string, err error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !parsed.Valid {
		return "", "", "", fmt.Errorf("invalid state")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] != oauthStateTyp {
		return "", "", "", fmt.Errorf("invalid state")
	}
	provider, _ = claims["p"].(string)
	mode, _ = claims["m"].(string)
	userID, _ = claims["uid"].(string)
	return provider, mode, userID, nil
}

// HandleCallback processes OAuth callback. For login mode returns session token if linked.
// For link mode requires matching session user.
func (s *OAuthService) HandleCallback(
	ctx context.Context, provider, code, state string, query map[string][]string, sessionUserID string,
) (rawSession string, err error) {
	p, mode, linkUID, err := s.parseState(state)
	if err != nil || p != provider {
		return "", fmt.Errorf("invalid oauth state")
	}
	providerUserID, email, err := s.exchange(ctx, provider, code, query)
	if err != nil {
		return "", err
	}
	switch mode {
	case "link":
		if sessionUserID == "" || sessionUserID != linkUID {
			return "", ErrForbidden
		}
		var em *string
		if email != "" {
			em = &email
		}
		return "", s.oauth.Link(ctx, sessionUserID, provider, providerUserID, em)
	case "login":
		ident, err := s.oauth.GetByProvider(ctx, provider, providerUserID)
		if err != nil {
			return "", ErrOAuthNotLinked
		}
		return s.auth.MintSession(ctx, ident.UserID)
	default:
		return "", fmt.Errorf("unknown oauth mode")
	}
}

func (s *OAuthService) Unlink(ctx context.Context, userID, provider string) error {
	hasPass, n, err := s.oauth.CountAuthMethods(ctx, userID)
	if err != nil {
		return err
	}
	if !hasPass && n <= 1 {
		return ErrLastAuthMethod
	}
	return s.oauth.Unlink(ctx, userID, provider)
}

func (s *OAuthService) ListLinked(ctx context.Context, userID string) ([]models.OAuthIdentity, error) {
	return s.oauth.ListByUser(ctx, userID)
}

func (s *OAuthService) exchange(ctx context.Context, provider, code string, query map[string][]string) (providerUserID, email string, err error) {
	switch provider {
	case "google":
		return s.exchangeGoogle(ctx, code)
	case "discord":
		return s.exchangeDiscord(ctx, code)
	case "steam":
		return s.exchangeSteam(ctx, query)
	default:
		return "", "", fmt.Errorf("unknown provider")
	}
}

func (s *OAuthService) exchangeGoogle(ctx context.Context, code string) (string, string, error) {
	if s.google == nil {
		return "", "", fmt.Errorf("google oauth not configured")
	}
	tok, err := s.google.Exchange(ctx, code)
	if err != nil {
		return "", "", err
	}
	client := s.google.Client(ctx, tok)
	res, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	var info struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return "", "", err
	}
	if info.Sub == "" {
		return "", "", fmt.Errorf("google userinfo missing sub")
	}
	return info.Sub, info.Email, nil
}

func (s *OAuthService) exchangeDiscord(ctx context.Context, code string) (string, string, error) {
	if s.discord == nil {
		return "", "", fmt.Errorf("discord oauth not configured")
	}
	tok, err := s.discord.Exchange(ctx, code)
	if err != nil {
		return "", "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
	if res.StatusCode >= 300 {
		return "", "", fmt.Errorf("discord userinfo: %s", res.Status)
	}
	var info struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "", "", err
	}
	if info.ID == "" {
		return "", "", fmt.Errorf("discord userinfo missing id")
	}
	return info.ID, info.Email, nil
}

func (s *OAuthService) exchangeSteam(ctx context.Context, query map[string][]string) (string, string, error) {
	if s.steamKey == "" {
		return "", "", fmt.Errorf("steam oauth not configured")
	}
	q := url.Values{}
	for k, vs := range query {
		if strings.HasPrefix(k, "openid.") {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
	}
	q.Set("openid.mode", "check_authentication")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://steamcommunity.com/openid/login", strings.NewReader(q.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	if !strings.Contains(string(body), "is_valid:true") {
		return "", "", fmt.Errorf("steam openid validation failed")
	}
	claimed := ""
	if vs := query["openid.claimed_id"]; len(vs) > 0 {
		claimed = vs[0]
	}
	const prefix = "https://steamcommunity.com/openid/id/"
	if !strings.HasPrefix(claimed, prefix) {
		return "", "", fmt.Errorf("invalid steam claimed_id")
	}
	steamID := strings.TrimPrefix(claimed, prefix)
	if steamID == "" {
		return "", "", fmt.Errorf("empty steam id")
	}
	return steamID, "", nil
}
