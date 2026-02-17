package config

// Config holds the server configuration.
type Config struct {
	ServerName      string
	ListenAddr      string
	DatabasePath    string
	MaxMessageSize  int
	RateLimitPerSec int
}
