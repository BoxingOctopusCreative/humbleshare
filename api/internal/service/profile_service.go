package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/boxingoctopus/humbleshare/api/internal/media"
	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
)

const (
	recentClaimsLimit = 12
	maxProfileSlugLen = 32
	minProfileSlugLen = 3
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrSlugTaken       = errors.New("profile slug already taken")

	profileSlugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,30}[a-z0-9]$`)

	reservedProfileSlugs = map[string]struct{}{
		"admin": {}, "api": {}, "auth": {}, "games": {}, "health": {},
		"invite": {}, "login": {}, "me": {}, "privacy": {}, "request-access": {},
		"settings": {}, "sitemap": {}, "terms": {}, "u": {},
	}
)

type ProfileService struct {
	users         repository.UserRepository
	codes         repository.CodeRepository
	userAssetsCDN string
}

func NewProfileService(users repository.UserRepository, codes repository.CodeRepository, userAssetsCDN string) *ProfileService {
	return &ProfileService{
		users:         users,
		codes:         codes,
		userAssetsCDN: strings.TrimRight(strings.TrimSpace(userAssetsCDN), "/"),
	}
}

// SoftSlugCandidate returns a base slug from preferred text (usually display name).
func SoftSlugCandidate(preferred string) string {
	base := SlugFromDisplayName(preferred)
	if base == "" {
		return "user"
	}
	return base
}

// SlugFromDisplayName turns a display name into a slug candidate.
// Short or empty results become "user".
func SlugFromDisplayName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "user"
	}
	var b strings.Builder
	lastHyphen := false
	for _, r := range name {
		switch {
		case r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(unicode.ToLower(r))
			lastHyphen = false
		default:
			if b.Len() > 0 && !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > maxProfileSlugLen {
		s = strings.Trim(s[:maxProfileSlugLen], "-")
	}
	if len(s) < minProfileSlugLen {
		return "user"
	}
	return s
}

// NormalizeProfileSlug lowercases and trims. Empty string is allowed (clears slug).
func NormalizeProfileSlug(slug string) (string, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return "", nil
	}
	if !profileSlugRE.MatchString(slug) {
		return "", ErrInvalidProfile
	}
	if _, reserved := reservedProfileSlugs[slug]; reserved {
		return "", ErrInvalidProfile
	}
	return slug, nil
}

// AllocateUniqueProfileSlug finds an unused slug based on preferred (usually display name).
func AllocateUniqueProfileSlug(ctx context.Context, users repository.UserRepository, preferred string) (string, error) {
	base := SoftSlugCandidate(preferred)
	for n := 0; n < 1000; n++ {
		candidate := slugCandidate(base, n)
		if candidate == "" {
			continue
		}
		if _, reserved := reservedProfileSlugs[candidate]; reserved {
			continue
		}
		if !profileSlugRE.MatchString(candidate) {
			continue
		}
		_, err := users.GetByProfileSlug(ctx, candidate)
		if errors.Is(err, repository.ErrUserNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", ErrInvalidProfile
}

func slugCandidate(base string, n int) string {
	if n == 0 {
		return base
	}
	suffix := fmt.Sprintf("-%d", n+1)
	maxBase := maxProfileSlugLen - len(suffix)
	if maxBase < 1 {
		return ""
	}
	trunc := base
	if len(trunc) > maxBase {
		trunc = strings.Trim(trunc[:maxBase], "-")
	}
	if trunc == "" {
		trunc = "u"
	}
	return trunc + suffix
}

func (s *ProfileService) GetBySlug(ctx context.Context, slug string, viewer *models.User) (*models.PublicProfile, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return nil, ErrProfileNotFound
	}
	u, err := s.users.GetByProfileSlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	isOwner := viewer != nil && viewer.ID == u.ID
	if !u.ProfilePublic && !isOwner {
		return nil, ErrProfileNotFound
	}

	claims, err := s.codes.ListRecentClaimsByUserID(ctx, u.ID, recentClaimsLimit)
	if err != nil {
		return nil, err
	}
	if claims == nil {
		claims = []models.PublicClaim{}
	}

	return &models.PublicProfile{
		ProfileSlug:   u.ProfileSlug,
		DisplayName:   u.DisplayName,
		Bio:           u.Bio,
		SteamGamertag: u.SteamGamertag,
		DiscordTag:    u.DiscordTag,
		AvatarURL:     media.ProxyURL(s.userAssetsCDN, u.AvatarURL),
		BannerURL:     media.ProxyURL(s.userAssetsCDN, u.BannerURL),
		ProfilePublic: u.ProfilePublic,
		RecentClaims:  claims,
		IsOwner:       isOwner,
	}, nil
}
