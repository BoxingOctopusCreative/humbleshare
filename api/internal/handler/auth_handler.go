package handler

import (
	"errors"
	"io"
	"net/url"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/httpx"
	"github.com/boxingoctopus/humbleshare/api/internal/middleware"
	"github.com/boxingoctopus/humbleshare/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	auth   *service.AuthService
	oauth  *service.OAuthService
	secure bool
	maxAge int
}

func NewAuthHandler(auth *service.AuthService, oauth *service.OAuthService, cookieSecure bool, sessionMaxAge int) *AuthHandler {
	return &AuthHandler{auth: auth, oauth: oauth, secure: cookieSecure, maxAge: sessionMaxAge}
}

func (h *AuthHandler) setSessionCookie(c *fiber.Ctx, raw string) {
	c.Cookie(&fiber.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    raw,
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: "Lax",
		MaxAge:   h.maxAge,
		Expires:  time.Now().Add(time.Duration(h.maxAge) * time.Second),
	})
}

func (h *AuthHandler) clearSessionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: "Lax",
		MaxAge:   -1,
		Expires:  time.Now().Add(-time.Hour),
	})
}

func mapAuthErr(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials),
		errors.Is(err, service.ErrInvalidPending),
		errors.Is(err, service.ErrInvalidTOTP):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	case errors.Is(err, service.ErrForbidden),
		errors.Is(err, service.ErrLastAuthMethod):
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrWeakPassword),
		errors.Is(err, service.ErrInvalidEmail),
		errors.Is(err, service.ErrRequestPending),
		errors.Is(err, service.ErrAlreadyRegistered),
		errors.Is(err, service.ErrInviteInvalid),
		errors.Is(err, service.ErrTwoFactorAlreadyOn),
		errors.Is(err, service.ErrOAuthNotLinked),
		errors.Is(err, service.ErrInvalidProfile),
		errors.Is(err, service.ErrInvalidAvatar):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrAvatarStorageUnset):
		return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
	default:
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
}

func (h *AuthHandler) RequestAccess(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
		Note  string `json:"note"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	ar, err := h.auth.RequestAccess(c.Context(), body.Email, body.Note)
	if err != nil {
		return mapAuthErr(err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":     ar.ID,
		"email":  ar.Email,
		"status": ar.Status,
	})
}

func (h *AuthHandler) GetInvite(c *fiber.Ctx) error {
	inv, err := h.auth.GetInvite(c.Context(), c.Params("token"))
	if err != nil {
		return mapAuthErr(err)
	}
	return c.JSON(fiber.Map{"email": inv.Email, "role": inv.Role, "expires_at": inv.ExpiresAt})
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var body struct {
		Token       string `json:"token"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	u, raw, err := h.auth.Register(c.Context(), body.Token, body.Password, body.DisplayName)
	if err != nil {
		return mapAuthErr(err)
	}
	h.setSessionCookie(c, raw)
	return c.Status(fiber.StatusCreated).JSON(u)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	res, err := h.auth.Login(c.Context(), body.Email, body.Password)
	if err != nil {
		return mapAuthErr(err)
	}
	if res.Requires2FA {
		return c.JSON(fiber.Map{"requires_2fa": true, "pending_token": res.Pending2FAToken})
	}
	h.setSessionCookie(c, res.RawSessionToken)
	return c.JSON(res.User)
}

func (h *AuthHandler) Login2FA(c *fiber.Ctx) error {
	var body struct {
		PendingToken string `json:"pending_token"`
		Code         string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	res, err := h.auth.CompleteLogin2FA(c.Context(), body.PendingToken, body.Code)
	if err != nil {
		return mapAuthErr(err)
	}
	h.setSessionCookie(c, res.RawSessionToken)
	return c.JSON(res.User)
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	raw, _ := c.Locals("session_token").(string)
	if raw == "" {
		raw = c.Cookies(middleware.SessionCookieName)
	}
	_ = h.auth.Logout(c.Context(), raw)
	h.clearSessionCookie(c)
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	u := middleware.UserFromLocals(c)
	return c.JSON(u)
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	u := middleware.UserFromLocals(c)
	var body struct {
		Current string `json:"current_password"`
		Next    string `json:"new_password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.auth.ChangePassword(c.Context(), u.ID, body.Current, body.Next); err != nil {
		return mapAuthErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	u := middleware.UserFromLocals(c)
	var body struct {
		DisplayName   string `json:"display_name"`
		Bio           string `json:"bio"`
		SteamGamertag string `json:"steam_gamertag"`
		DiscordTag    string `json:"discord_tag"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	updated, err := h.auth.UpdateProfile(c.Context(), u.ID, body.DisplayName, body.Bio, body.SteamGamertag, body.DiscordTag)
	if err != nil {
		return mapAuthErr(err)
	}
	return c.JSON(updated)
}

func (h *AuthHandler) UploadAvatar(c *fiber.Ctx) error {
	u := middleware.UserFromLocals(c)
	fh, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file is required")
	}
	f, err := fh.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "unable to read file")
	}
	defer f.Close()
	const maxRead = 2*1024*1024 + 1
	buf := make([]byte, maxRead)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return fiber.NewError(fiber.StatusBadRequest, "unable to read file")
	}
	data := buf[:n]
	updated, err := h.auth.UploadAvatar(c.Context(), u.ID, data)
	if err != nil {
		return mapAuthErr(err)
	}
	return c.JSON(updated)
}

func (h *AuthHandler) DeleteAvatar(c *fiber.Ctx) error {
	u := middleware.UserFromLocals(c)
	updated, err := h.auth.DeleteAvatar(c.Context(), u.ID)
	if err != nil {
		return mapAuthErr(err)
	}
	return c.JSON(updated)
}

func (h *AuthHandler) Setup2FA(c *fiber.Ctx) error {
	u := middleware.UserFromLocals(c)
	res, err := h.auth.SetupTwoFactor(c.Context(), u.ID)
	if err != nil {
		return mapAuthErr(err)
	}
	return c.JSON(res)
}

func (h *AuthHandler) Enable2FA(c *fiber.Ctx) error {
	u := middleware.UserFromLocals(c)
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.auth.EnableTwoFactor(c.Context(), u.ID, body.Code); err != nil {
		return mapAuthErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AuthHandler) Disable2FA(c *fiber.Ctx) error {
	u := middleware.UserFromLocals(c)
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.auth.DisableTwoFactor(c.Context(), u.ID, body.Code); err != nil {
		return mapAuthErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AuthHandler) OAuthStart(c *fiber.Ctx) error {
	provider := c.Params("provider")
	switch provider {
	case "google", "discord", "steam":
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unknown provider")
	}
	mode := c.Query("mode", "login")
	if mode != "login" && mode != "link" {
		return fiber.NewError(fiber.StatusBadRequest, "invalid mode")
	}
	userID := ""
	if mode == "link" {
		u := middleware.UserFromLocals(c)
		if u == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		userID = u.ID
	}
	authURL, err := h.oauth.AuthURL(provider, mode, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if !oauthAuthorizeURLAllowed(authURL) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid oauth authorize url")
	}
	// Return URL for the BFF/client to navigate; avoid server-side open redirect on request input.
	return c.JSON(fiber.Map{"authorize_url": authURL})
}

func oauthAuthorizeURLAllowed(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return false
	}
	switch u.Host {
	case "accounts.google.com", "discord.com", "steamcommunity.com":
		return true
	default:
		return false
	}
}

func (h *AuthHandler) OAuthCallback(c *fiber.Ctx) error {
	provider := c.Params("provider")
	state := c.Query("state")
	code := c.Query("code")
	sessionUID := ""
	if raw := c.Cookies(middleware.SessionCookieName); raw != "" {
		if u, err := h.auth.UserFromSession(c.Context(), raw); err == nil {
			sessionUID = u.ID
		}
	}
	q := mapToValues(c)
	raw, err := h.oauth.HandleCallback(c.Context(), provider, code, state, q, sessionUID)
	if err != nil {
		return mapAuthErr(err)
	}
	if raw != "" {
		h.setSessionCookie(c, raw)
		return c.Redirect("/", fiber.StatusFound)
	}
	return c.Redirect("/settings", fiber.StatusFound)
}

func mapToValues(c *fiber.Ctx) map[string][]string {
	out := make(map[string][]string)
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		key := string(k)
		out[key] = append(out[key], string(v))
	})
	return out
}

func (h *AuthHandler) ListOAuth(c *fiber.Ctx) error {
	u := middleware.UserFromLocals(c)
	list, err := h.oauth.ListLinked(c.Context(), u.ID)
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	return c.JSON(list)
}

func (h *AuthHandler) UnlinkOAuth(c *fiber.Ctx) error {
	u := middleware.UserFromLocals(c)
	if err := h.oauth.Unlink(c.Context(), u.ID, c.Params("provider")); err != nil {
		return mapAuthErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AuthHandler) ListAccessRequests(c *fiber.Ctx) error {
	list, err := h.auth.ListAccessRequests(c.Context())
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	return c.JSON(list)
}

func (h *AuthHandler) ApproveAccessRequest(c *fiber.Ctx) error {
	id := c.Params("id")
	token := c.Query("token")
	actor := middleware.UserFromLocals(c)
	_, _, err := h.auth.ApproveRequest(c.Context(), id, actor, token)
	if err != nil {
		return mapAuthErr(err)
	}
	if c.Method() == fiber.MethodGet {
		return c.Redirect("/admin?tab=requests", fiber.StatusFound)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AuthHandler) DenyAccessRequest(c *fiber.Ctx) error {
	actor := middleware.UserFromLocals(c)
	if err := h.auth.DenyRequest(c.Context(), c.Params("id"), actor); err != nil {
		return mapAuthErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AuthHandler) ListUsers(c *fiber.Ctx) error {
	list, err := h.auth.ListUsers(c.Context())
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	return c.JSON(list)
}

func (h *AuthHandler) SetUserRole(c *fiber.Ctx) error {
	actor := middleware.UserFromLocals(c)
	var body struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.auth.SetUserRole(c.Context(), actor, c.Params("id"), body.Role); err != nil {
		return mapAuthErr(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}
