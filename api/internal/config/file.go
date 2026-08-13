package config

// fileConfig is the optional TOML file shape; all fields are optional.
type fileConfig struct {
	Server   fileServer   `toml:"server"`
	Database fileDatabase `toml:"database"`
	CORS     fileCORS     `toml:"cors"`
	Auth      fileAuth      `toml:"auth"`
	Mailgun   fileMailgun   `toml:"mailgun"`
	SMTP      fileSMTP      `toml:"smtp"`
	OAuth     fileOAuth     `toml:"oauth"`
	Turnstile fileTurnstile `toml:"turnstile"`
	Steam     fileSteam     `toml:"steam"`
	R2        fileR2        `toml:"r2"`
	Sentry    fileSentry    `toml:"sentry"`
}

type fileServer struct {
	HTTPAddr string `toml:"http_addr"`
}

type fileDatabase struct {
	URL string `toml:"url"`
}

type fileCORS struct {
	Origins []string `toml:"origins"`
}

type fileAuth struct {
	JWTSecret          string `toml:"jwt_secret"`
	SuperadminEmail    string `toml:"superadmin_email"`
	SuperadminPassword string `toml:"superadmin_password"`
	CookieSecure       *bool  `toml:"cookie_secure"`
	SessionMaxAge      int    `toml:"session_max_age_seconds"`
	PublicWebBaseURL   string `toml:"public_web_base_url"`
}

type fileMailgun struct {
	APIKey  string `toml:"api_key"`
	Domain  string `toml:"domain"`
	From    string `toml:"from"`
	APIBase string `toml:"api_base"`
}

type fileSMTP struct {
	Host string `toml:"host"`
	Port string `toml:"port"`
	From string `toml:"from"`
}

type fileOAuth struct {
	GoogleClientID       string `toml:"google_client_id"`
	GoogleClientSecret   string `toml:"google_client_secret"`
	DiscordClientID      string `toml:"discord_client_id"`
	DiscordClientSecret  string `toml:"discord_client_secret"`
	SteamAPIKey          string `toml:"steam_api_key"`
}

type fileTurnstile struct {
	SiteKey   string `toml:"site_key"`
	SecretKey string `toml:"secret_key"`
}

type fileSteam struct {
	MetadataCacheTTLSeconds int     `toml:"metadata_cache_ttl_seconds"`
	RatePerSecond           float64 `toml:"rate_per_second"`
	ClaimRateLimitPerMinute int     `toml:"claim_rate_limit_per_minute"`
}

type fileR2 struct {
	AccountID               string `toml:"account_id"`
	AccessKeyID             string `toml:"access_key_id"`
	SecretAccessKey         string `toml:"secret_access_key"`
	Bucket                  string `toml:"bucket"`
	PublicBaseURL           string `toml:"public_base_url"`
	UserAssetsBucket        string `toml:"user_assets_bucket"`
	UserAssetsPublicBaseURL string `toml:"user_assets_public_base_url"`
}

type fileSentry struct {
	DSN              string  `toml:"dsn"`
	Environment      string  `toml:"environment"`
	Release          string  `toml:"release"`
	TracesSampleRate float64 `toml:"traces_sample_rate"`
}
