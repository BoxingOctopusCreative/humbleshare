package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/mailer"
	"github.com/boxingoctopus/humbleshare/api/internal/media"
	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/boxingoctopus/humbleshare/api/internal/storage"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost        = 12
	twoFAPendingTyp   = "2fa_pending"
	approveRequestTyp = "approve_access"
	inviteTTL         = 7 * 24 * time.Hour
	twoFAPendingTTL   = 5 * time.Minute
	approveTokenTTL   = 7 * 24 * time.Hour

	maxDisplayNameLen = 64
	maxBioLen         = 500
	maxGamertagLen    = 64
	maxAvatarBytes    = 2 * 1024 * 1024
	maxBannerBytes    = 4 * 1024 * 1024
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrInvalidTOTP        = errors.New("invalid two-factor code")
	ErrInvalidPending     = errors.New("invalid or expired login token")
	ErrTwoFactorAlreadyOn = errors.New("two-factor authentication is already enabled")
	ErrInviteInvalid      = errors.New("invite is invalid or expired")
	ErrRequestPending     = errors.New("an access request for this email is already pending")
	ErrAlreadyRegistered  = errors.New("an account with this email already exists")
	ErrForbidden          = errors.New("forbidden")
	ErrLastAuthMethod     = errors.New("cannot remove the last sign-in method")
	ErrOAuthNotLinked     = errors.New("this provider is not linked; link it from account settings first")
	ErrInvalidProfile     = errors.New("invalid profile fields")
	ErrInvalidAvatar      = errors.New("avatar must be a jpeg, png, or webp under 2MB")
	ErrInvalidBanner      = errors.New("banner must be a jpeg, png, or webp under 4MB")
	ErrAvatarStorageUnset = errors.New("avatar storage is not configured")
)

type LoginResult struct {
	User            *models.User `json:"user,omitempty"`
	RawSessionToken string       `json:"-"`
	Pending2FAToken string       `json:"pending_token,omitempty"`
	Requires2FA     bool         `json:"requires_2fa"`
}

type TwoFASetupResult struct {
	Secret     string `json:"secret"`
	OtpauthURL string `json:"otpauth_url"`
}

type AuthService struct {
	users               repository.UserRepository
	sessions            repository.SessionRepository
	requests            repository.AccessRequestRepository
	invites             repository.AccessInviteRepository
	oauth               repository.OAuthIdentityRepository
	mail                mailer.Sender
	objects             storage.ObjectStore
	jwtSecret           []byte
	sessionTTL          time.Duration
	publicWebBaseURL    string
	userAssetsPublicURL string
	superadminEmail     string
	superadminPassword  string
}

func NewAuthService(
	users repository.UserRepository,
	sessions repository.SessionRepository,
	requests repository.AccessRequestRepository,
	invites repository.AccessInviteRepository,
	oauth repository.OAuthIdentityRepository,
	mail mailer.Sender,
	objects storage.ObjectStore,
	jwtSecret, publicWebBaseURL, superadminEmail, superadminPassword, userAssetsPublicURL string,
	sessionMaxAgeSeconds int,
) *AuthService {
	if sessionMaxAgeSeconds < 300 {
		sessionMaxAgeSeconds = 30 * 24 * 3600
	}
	return &AuthService{
		users:               users,
		sessions:            sessions,
		requests:            requests,
		invites:             invites,
		oauth:               oauth,
		mail:                mail,
		objects:             objects,
		jwtSecret:           []byte(jwtSecret),
		sessionTTL:          time.Duration(sessionMaxAgeSeconds) * time.Second,
		publicWebBaseURL:    strings.TrimRight(strings.TrimSpace(publicWebBaseURL), "/"),
		userAssetsPublicURL: strings.TrimRight(strings.TrimSpace(userAssetsPublicURL), "/"),
		superadminEmail:     strings.TrimSpace(strings.ToLower(superadminEmail)),
		superadminPassword:  superadminPassword,
	}
}

func normalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", ErrInvalidEmail
	}
	return email, nil
}

func (s *AuthService) RequestAccess(ctx context.Context, email, note string) (*models.AccessRequest, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, ErrAlreadyRegistered
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, err
	}
	pending, err := s.requests.HasPending(ctx, email)
	if err != nil {
		return nil, err
	}
	if pending {
		return nil, ErrRequestPending
	}
	ar, err := s.requests.Create(ctx, email, strings.TrimSpace(note))
	if err != nil {
		return nil, err
	}
	s.notifyAdminsOfRequest(ctx, ar)
	return ar, nil
}

func (s *AuthService) notifyAdminsOfRequest(ctx context.Context, ar *models.AccessRequest) {
	emails, err := s.users.ListAdminEmails(ctx)
	if err != nil {
		log.Printf("access request notify: list admins: %v", err)
		emails = nil
	}
	if s.superadminEmail != "" {
		found := false
		for _, e := range emails {
			if strings.EqualFold(e, s.superadminEmail) {
				found = true
				break
			}
		}
		if !found {
			emails = append(emails, s.superadminEmail)
		}
	}
	approveURL := ""
	if tok, err := s.signApproveToken(ar.ID); err == nil && s.publicWebBaseURL != "" {
		approveURL = s.publicWebBaseURL + "/admin/access-requests?approve=" + ar.ID + "&token=" + tok
	}
	body := fmt.Sprintf(
		"Access request for HumbleShare\n\nEmail: %s\nNote: %s\nRequest ID: %s\n",
		ar.Email, ar.Note, ar.ID,
	)
	if approveURL != "" {
		body += "\nApprove: " + approveURL + "\n"
	}
	for _, to := range emails {
		if s.mail == nil {
			log.Printf("access request notify: mailer unset; would email %s request %s", to, ar.ID)
			continue
		}
		if err := s.mail.Send(ctx, mailer.Message{
			To:      to,
			Subject: "HumbleShare access request",
			Text:    body,
		}); err != nil {
			log.Printf("access request notify: send to %s: %v", to, err)
		}
	}
}

func (s *AuthService) signApproveToken(requestID string) (string, error) {
	claims := jwt.MapClaims{
		"typ": approveRequestTyp,
		"rid": requestID,
		"exp": time.Now().Add(approveTokenTTL).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *AuthService) ApproveRequest(ctx context.Context, requestID string, actor *models.User, approveJWT string) (*models.AccessInvite, string, error) {
	ar, err := s.requests.GetByID(ctx, requestID)
	if err != nil {
		return nil, "", err
	}
	if ar.Status != "pending" {
		return nil, "", fmt.Errorf("request is not pending")
	}
	if actor == nil {
		if err := s.verifyApproveToken(approveJWT, requestID); err != nil {
			return nil, "", ErrForbidden
		}
	} else if !actor.IsAdmin() {
		return nil, "", ErrForbidden
	}
	var resolvedBy *string
	if actor != nil {
		resolvedBy = &actor.ID
	}
	if err := s.requests.Resolve(ctx, requestID, "approved", resolvedBy); err != nil {
		return nil, "", err
	}
	return s.createAndSendInvite(ctx, ar.Email, models.RoleMember, resolvedBy, &ar.ID)
}

func (s *AuthService) DenyRequest(ctx context.Context, requestID string, actor *models.User) error {
	if actor == nil || !actor.IsAdmin() {
		return ErrForbidden
	}
	return s.requests.Resolve(ctx, requestID, "denied", &actor.ID)
}

func (s *AuthService) verifyApproveToken(token, requestID string) error {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !parsed.Valid {
		return ErrForbidden
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] != approveRequestTyp {
		return ErrForbidden
	}
	rid, _ := claims["rid"].(string)
	if rid != requestID {
		return ErrForbidden
	}
	return nil
}

func (s *AuthService) createAndSendInvite(
	ctx context.Context, email, role string, createdBy, requestID *string,
) (*models.AccessInvite, string, error) {
	raw, hash, err := newSessionTokenPair()
	if err != nil {
		return nil, "", err
	}
	inv, err := s.invites.Create(ctx, email, hash, role, time.Now().Add(inviteTTL), createdBy, requestID)
	if err != nil {
		return nil, "", err
	}
	link := ""
	if s.publicWebBaseURL != "" {
		link = s.publicWebBaseURL + "/invite/" + raw
	}
	text, html := mailer.InviteEmail(link, s.publicWebBaseURL)
	if s.mail == nil {
		log.Printf("invite: mailer unset; invite URL for %s: %s", email, link)
	} else if err := s.mail.Send(ctx, mailer.Message{
		To:      email,
		Subject: "Your HumbleShare invite",
		Text:    text,
		HTML:    html,
	}); err != nil {
		return inv, raw, fmt.Errorf("invite created but email failed: %w", err)
	}
	return inv, raw, nil
}

// EnsureSuperadminInvite creates the first superadmin when missing.
// If SUPERADMIN_PASSWORD is set, the account is created directly; otherwise an invite is emailed/logged.
func (s *AuthService) EnsureSuperadminInvite(ctx context.Context) error {
	if s.superadminEmail == "" {
		return nil
	}
	if _, err := s.users.GetByEmail(ctx, s.superadminEmail); err == nil {
		return nil
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		return err
	}

	if pw := strings.TrimSpace(s.superadminPassword); pw != "" {
		if len(pw) < 8 {
			return fmt.Errorf("SUPERADMIN_PASSWORD must be at least 8 characters")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcryptCost)
		if err != nil {
			return err
		}
		dn := strings.Split(s.superadminEmail, "@")[0]
		slug, err := AllocateUniqueProfileSlug(ctx, s.users, dn)
		if err != nil {
			return err
		}
		if _, err := s.users.Create(ctx, s.superadminEmail, string(hash), dn, models.RoleSuperadmin, slug); err != nil {
			return err
		}
		log.Printf("superadmin: created account for %s from SUPERADMIN_PASSWORD", s.superadminEmail)
		return nil
	}

	open, err := s.invites.HasOpenForEmail(ctx, s.superadminEmail)
	if err != nil {
		return err
	}
	if open {
		return nil
	}
	_, _, err = s.createAndSendInvite(ctx, s.superadminEmail, models.RoleSuperadmin, nil, nil)
	if err != nil {
		log.Printf("superadmin invite: %v", err)
	} else {
		log.Printf("superadmin invite: sent/logged for %s", s.superadminEmail)
	}
	return nil
}

func (s *AuthService) GetInvite(ctx context.Context, rawToken string) (*models.AccessInvite, error) {
	inv, err := s.invites.GetByTokenHash(ctx, hashToken(rawToken))
	if err != nil {
		return nil, ErrInviteInvalid
	}
	if inv.ConsumedAt != nil || time.Now().After(inv.ExpiresAt) {
		return nil, ErrInviteInvalid
	}
	return inv, nil
}

func (s *AuthService) Register(ctx context.Context, rawToken, password, displayName string) (*models.User, string, error) {
	if len(password) < 8 {
		return nil, "", ErrWeakPassword
	}
	inv, err := s.GetInvite(ctx, rawToken)
	if err != nil {
		return nil, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, "", err
	}
	dn := strings.TrimSpace(displayName)
	if dn == "" {
		dn = strings.Split(inv.Email, "@")[0]
	}
	slug, err := AllocateUniqueProfileSlug(ctx, s.users, dn)
	if err != nil {
		return nil, "", err
	}
	u, err := s.users.Create(ctx, inv.Email, string(hash), dn, inv.Role, slug)
	if err != nil {
		return nil, "", err
	}
	if err := s.invites.Consume(ctx, inv.ID); err != nil {
		return nil, "", err
	}
	rawSession, err := s.MintSession(ctx, u.ID)
	if err != nil {
		return u, "", err
	}
	return u, rawSession, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	if u.TwoFactorEnabled {
		tok, err := s.sign2FAPending(u.ID)
		if err != nil {
			return nil, err
		}
		return &LoginResult{Requires2FA: true, Pending2FAToken: tok}, nil
	}
	raw, err := s.MintSession(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: u, RawSessionToken: raw}, nil
}

func (s *AuthService) CompleteLogin2FA(ctx context.Context, pendingToken, code string) (*LoginResult, error) {
	userID, err := s.parse2FAPending(pendingToken)
	if err != nil {
		return nil, ErrInvalidPending
	}
	secret, err := s.users.GetTwoFactorSecret(ctx, userID)
	if err != nil {
		return nil, ErrInvalidPending
	}
	if !totp.Validate(strings.TrimSpace(code), secret) {
		return nil, ErrInvalidTOTP
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	raw, err := s.MintSession(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: u, RawSessionToken: raw}, nil
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.sessions.DeleteByTokenHash(ctx, hashToken(rawToken))
}

func (s *AuthService) UserFromSession(ctx context.Context, rawToken string) (*models.User, error) {
	sess, err := s.sessions.GetByTokenHash(ctx, hashToken(rawToken))
	if err != nil {
		return nil, err
	}
	u, err := s.users.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}
	return s.EnsureProfileSlug(ctx, u)
}

// EnsureProfileSlug assigns a display-name-based slug when the user has none yet.
func (s *AuthService) EnsureProfileSlug(ctx context.Context, u *models.User) (*models.User, error) {
	if u == nil || strings.TrimSpace(u.ProfileSlug) != "" {
		return u, nil
	}
	slug, err := AllocateUniqueProfileSlug(ctx, s.users, u.DisplayName)
	if err != nil {
		return nil, err
	}
	if err := s.users.UpdateProfile(ctx, u.ID, u.DisplayName, u.Bio, u.SteamGamertag, u.DiscordTag, slug, u.ProfilePublic); err != nil {
		if errors.Is(err, repository.ErrProfileSlugTaken) {
			slug, err = AllocateUniqueProfileSlug(ctx, s.users, u.DisplayName)
			if err != nil {
				return nil, err
			}
			if err := s.users.UpdateProfile(ctx, u.ID, u.DisplayName, u.Bio, u.SteamGamertag, u.DiscordTag, slug, u.ProfilePublic); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return s.users.GetByID(ctx, u.ID)
}

// MintSession creates an opaque session and returns the raw cookie/bearer token.
func (s *AuthService) MintSession(ctx context.Context, userID string) (string, error) {
	raw, hash, err := newSessionTokenPair()
	if err != nil {
		return "", err
	}
	_, err = s.sessions.Create(ctx, userID, hash, time.Now().Add(s.sessionTTL))
	if err != nil {
		return "", err
	}
	return raw, nil
}

func (s *AuthService) sign2FAPending(userID string) (string, error) {
	claims := jwt.MapClaims{
		"typ": twoFAPendingTyp,
		"uid": userID,
		"exp": time.Now().Add(twoFAPendingTTL).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *AuthService) parse2FAPending(token string) (string, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !parsed.Valid {
		return "", ErrInvalidPending
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] != twoFAPendingTyp {
		return "", ErrInvalidPending
	}
	uid, _ := claims["uid"].(string)
	if uid == "" {
		return "", ErrInvalidPending
	}
	return uid, nil
}

func (s *AuthService) SetupTwoFactor(ctx context.Context, userID string) (*TwoFASetupResult, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.TwoFactorEnabled {
		return nil, ErrTwoFactorAlreadyOn
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "HumbleShare",
		AccountName: u.Email,
	})
	if err != nil {
		return nil, err
	}
	if err := s.users.SetTwoFactorSecret(ctx, userID, key.Secret()); err != nil {
		return nil, err
	}
	return &TwoFASetupResult{Secret: key.Secret(), OtpauthURL: key.URL()}, nil
}

func (s *AuthService) EnableTwoFactor(ctx context.Context, userID, code string) error {
	secret, err := s.users.GetTwoFactorSecret(ctx, userID)
	if err != nil {
		return err
	}
	if !totp.Validate(strings.TrimSpace(code), secret) {
		return ErrInvalidTOTP
	}
	return s.users.EnableTwoFactor(ctx, userID)
}

func (s *AuthService) DisableTwoFactor(ctx context.Context, userID, code string) error {
	secret, err := s.users.GetTwoFactorSecret(ctx, userID)
	if err != nil {
		return ErrInvalidTOTP
	}
	if !totp.Validate(strings.TrimSpace(code), secret) {
		return ErrInvalidTOTP
	}
	return s.users.DisableTwoFactor(ctx, userID)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID, current, next string) error {
	if len(next) < 8 {
		return ErrWeakPassword
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(current)); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcryptCost)
	if err != nil {
		return err
	}
	return s.users.UpdatePassword(ctx, userID, string(hash))
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID, displayName, bio, steamGamertag, discordTag, profileSlug string, profilePublic bool) (*models.User, error) {
	displayName = strings.TrimSpace(displayName)
	bio = strings.TrimSpace(bio)
	steamGamertag = strings.TrimSpace(steamGamertag)
	discordTag = strings.TrimSpace(discordTag)
	if displayName == "" {
		return nil, ErrInvalidProfile
	}
	if len(displayName) > maxDisplayNameLen || len(bio) > maxBioLen ||
		len(steamGamertag) > maxGamertagLen || len(discordTag) > maxGamertagLen {
		return nil, ErrInvalidProfile
	}
	slug, err := NormalizeProfileSlug(profileSlug)
	if err != nil {
		return nil, err
	}
	if slug == "" {
		slug, err = AllocateUniqueProfileSlug(ctx, s.users, displayName)
		if err != nil {
			return nil, err
		}
	}
	if profilePublic && slug == "" {
		return nil, ErrInvalidProfile
	}
	if err := s.users.UpdateProfile(ctx, userID, displayName, bio, steamGamertag, discordTag, slug, profilePublic); err != nil {
		if errors.Is(err, repository.ErrProfileSlugTaken) {
			return nil, ErrSlugTaken
		}
		return nil, err
	}
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) UploadAvatar(ctx context.Context, userID string, data []byte) (*models.User, error) {
	if s.objects == nil || s.userAssetsPublicURL == "" {
		return nil, ErrAvatarStorageUnset
	}
	if len(data) == 0 || len(data) > maxAvatarBytes {
		return nil, ErrInvalidAvatar
	}
	contentType, ext, ok := sniffImage(data)
	if !ok {
		return nil, ErrInvalidAvatar
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("avatars/%s.%s", userID, ext)
	if err := s.objects.Put(ctx, key, bytes.NewReader(data), contentType, int64(len(data))); err != nil {
		return nil, err
	}
	if oldKey := objectKeyFromUserAssetURL(s.userAssetsPublicURL, u.AvatarURL, "avatars/"); oldKey != "" && oldKey != key {
		_ = s.objects.Delete(ctx, oldKey)
	}
	avatarURL := s.userAssetsPublicURL + "/" + key
	if err := s.users.UpdateAvatarURL(ctx, userID, avatarURL); err != nil {
		return nil, err
	}
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) DeleteAvatar(ctx context.Context, userID string) (*models.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.AvatarURL == "" {
		return u, nil
	}
	if s.objects != nil {
		if key := objectKeyFromUserAssetURL(s.userAssetsPublicURL, u.AvatarURL, "avatars/"); key != "" {
			_ = s.objects.Delete(ctx, key)
		}
	}
	if err := s.users.UpdateAvatarURL(ctx, userID, ""); err != nil {
		return nil, err
	}
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) UploadBanner(ctx context.Context, userID string, data []byte) (*models.User, error) {
	if s.objects == nil || s.userAssetsPublicURL == "" {
		return nil, ErrAvatarStorageUnset
	}
	if len(data) == 0 || len(data) > maxBannerBytes {
		return nil, ErrInvalidBanner
	}
	contentType, ext, ok := sniffImage(data)
	if !ok {
		return nil, ErrInvalidBanner
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("banners/%s.%s", userID, ext)
	if err := s.objects.Put(ctx, key, bytes.NewReader(data), contentType, int64(len(data))); err != nil {
		return nil, err
	}
	if oldKey := objectKeyFromUserAssetURL(s.userAssetsPublicURL, u.BannerURL, "banners/"); oldKey != "" && oldKey != key {
		_ = s.objects.Delete(ctx, oldKey)
	}
	bannerURL := s.userAssetsPublicURL + "/" + key
	if err := s.users.UpdateBannerURL(ctx, userID, bannerURL); err != nil {
		return nil, err
	}
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) DeleteBanner(ctx context.Context, userID string) (*models.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.BannerURL == "" {
		return u, nil
	}
	if s.objects != nil {
		if key := objectKeyFromUserAssetURL(s.userAssetsPublicURL, u.BannerURL, "banners/"); key != "" {
			_ = s.objects.Delete(ctx, key)
		}
	}
	if err := s.users.UpdateBannerURL(ctx, userID, ""); err != nil {
		return nil, err
	}
	return s.users.GetByID(ctx, userID)
}

// PresentUser returns a copy with avatar/banner URLs rewritten to the same-origin media proxy.
func (s *AuthService) PresentUser(u *models.User) *models.User {
	if u == nil {
		return nil
	}
	out := *u
	out.AvatarURL = media.ProxyURL(s.userAssetsPublicURL, u.AvatarURL)
	out.BannerURL = media.ProxyURL(s.userAssetsPublicURL, u.BannerURL)
	return &out
}

func sniffImage(data []byte) (contentType, ext string, ok bool) {
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return "image/jpeg", "jpg", true
	}
	if len(data) >= 8 &&
		data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4e && data[3] == 0x47 &&
		data[4] == 0x0d && data[5] == 0x0a && data[6] == 0x1a && data[7] == 0x0a {
		return "image/png", "png", true
	}
	if len(data) >= 12 &&
		string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp", "webp", true
	}
	return "", "", false
}

func objectKeyFromUserAssetURL(publicBase, assetURL, requiredPrefix string) string {
	assetURL = strings.TrimSpace(assetURL)
	if assetURL == "" {
		return ""
	}
	if u, err := url.Parse(assetURL); err == nil {
		key := strings.TrimPrefix(u.Path, "/")
		if userAssetKeyOK(key, requiredPrefix) {
			return key
		}
	}
	publicBase = strings.TrimRight(strings.TrimSpace(publicBase), "/")
	if publicBase != "" && strings.HasPrefix(assetURL, publicBase+"/") {
		key := strings.TrimPrefix(assetURL, publicBase+"/")
		if userAssetKeyOK(key, requiredPrefix) {
			return key
		}
	}
	return ""
}

func userAssetKeyOK(key, requiredPrefix string) bool {
	if key == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return false
	}
	return strings.HasPrefix(key, requiredPrefix)
}

// objectKeyFromAvatarURL is kept for tests; prefers avatars/ prefix.
func objectKeyFromAvatarURL(publicBase, avatarURL string) string {
	return objectKeyFromUserAssetURL(publicBase, avatarURL, "avatars/")
}

func (s *AuthService) SetUserRole(ctx context.Context, actor *models.User, targetID, role string) error {
	if actor == nil || !actor.IsSuperadmin() {
		return ErrForbidden
	}
	if role != models.RoleMember && role != models.RoleAdmin {
		return fmt.Errorf("role must be member or admin")
	}
	target, err := s.users.GetByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.IsSuperadmin() || strings.EqualFold(target.Email, s.superadminEmail) {
		return ErrForbidden
	}
	return s.users.UpdateRole(ctx, targetID, role)
}

func (s *AuthService) SuperadminEmail() string { return s.superadminEmail }

func (s *AuthService) ListUsers(ctx context.Context) ([]models.User, error) {
	return s.users.List(ctx)
}

func (s *AuthService) ListAccessRequests(ctx context.Context) ([]models.AccessRequest, error) {
	return s.requests.List(ctx)
}

func (s *AuthService) ListPendingRequests(ctx context.Context) ([]models.AccessRequest, error) {
	return s.requests.ListPending(ctx)
}
