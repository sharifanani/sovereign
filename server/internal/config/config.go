package config

// Config holds the server configuration.
type Config struct {
	ServerName      string
	ListenAddr      string
	DatabasePath    string
	MaxMessageSize  int
	RateLimitPerSec int

	// WebAuthn configuration
	RPDisplayName string   // Relying Party display name
	RPID          string   // Relying Party ID (domain)
	RPOrigins     []string // Allowed origins for WebAuthn ceremonies
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ServerName:      "sovereign",
		ListenAddr:      ":8080",
		DatabasePath:    "sovereign.db",
		MaxMessageSize:  65536, // 64KB
		RateLimitPerSec: 30,
		RPDisplayName:   "Sovereign",
		RPID:            "localhost",
		RPOrigins:       []string{"http://localhost:8080"},
	}
}
