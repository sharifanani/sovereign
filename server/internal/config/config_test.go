package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	tests := []struct {
		name string
		get  func(Config) any
		want any
	}{
		{
			name: "ServerName",
			get:  func(c Config) any { return c.ServerName },
			want: "sovereign",
		},
		{
			name: "ListenAddr",
			get:  func(c Config) any { return c.ListenAddr },
			want: ":8080",
		},
		{
			name: "DatabasePath",
			get:  func(c Config) any { return c.DatabasePath },
			want: "sovereign.db",
		},
		{
			name: "MaxMessageSize",
			get:  func(c Config) any { return c.MaxMessageSize },
			want: 65536,
		},
		{
			name: "RateLimitPerSec",
			get:  func(c Config) any { return c.RateLimitPerSec },
			want: 30,
		},
	}

	cfg := DefaultConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.get(cfg); got != tt.want {
				t.Errorf("DefaultConfig().%s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestDefaultConfigNoZeroValues(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ServerName == "" {
		t.Error("ServerName is empty")
	}
	if cfg.ListenAddr == "" {
		t.Error("ListenAddr is empty")
	}
	if cfg.DatabasePath == "" {
		t.Error("DatabasePath is empty")
	}
	if cfg.MaxMessageSize == 0 {
		t.Error("MaxMessageSize is zero")
	}
	if cfg.RateLimitPerSec == 0 {
		t.Error("RateLimitPerSec is zero")
	}
}
