package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type TelegramConfig struct {
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

type WebhookConfig struct {
	Discord  string         `yaml:"discord"`
	Telegram TelegramConfig `yaml:"telegram"`
}

type EmailConfig struct {
	Enabled     bool     `yaml:"enabled"`
	SMTPHost    string   `yaml:"smtp_host"`
	SMTPPort    int      `yaml:"smtp_port"`
	SMTPUser    string   `yaml:"smtp_username"`
	SMTPPass    string   `yaml:"smtp_password"`
	FromAddress string   `yaml:"from_address"`
	ToAddresses []string `yaml:"to_addresses"`
}

type TwilioConfig struct {
	Enabled    bool     `yaml:"enabled"`
	AccountSID string   `yaml:"account_sid"`
	AuthToken  string   `yaml:"auth_token"`
	FromPhone  string   `yaml:"from_phone"`
	ToPhones   []string `yaml:"to_phones"`
}

type AlertsConfig struct {
	Email  EmailConfig  `yaml:"email"`
	Twilio TwilioConfig `yaml:"twilio"`
}

type DashboardConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

type StorageConfig struct {
	DBPath string `yaml:"db_path"`
}

type Target struct {
	Name             string            `yaml:"name"`
	URL              string            `yaml:"url"`
	Method           string            `yaml:"method"`
	ExpectedStatus   int               `yaml:"expected_status"`
	AllowedStatuses  []int             `yaml:"allowed_statuses"`
	Headers          map[string]string `yaml:"headers"`
	TimeoutSeconds   int               `yaml:"timeout_seconds"`
}

type Config struct {
	IntervalSeconds  int             `yaml:"interval_seconds"`
	FailureThreshold int             `yaml:"failure_threshold"`
	Webhooks         WebhookConfig   `yaml:"webhooks"`
	Alerts           AlertsConfig    `yaml:"alerts"`
	Dashboard        DashboardConfig `yaml:"dashboard"`
	Storage          StorageConfig   `yaml:"storage"`
	Targets          []Target        `yaml:"targets"`
}

func (t Target) IsStatusAllowed(code int) bool {
	if len(t.AllowedStatuses) > 0 {
		for _, s := range t.AllowedStatuses {
			if code == s {
				return true
			}
		}
		return false
	}
	return code == t.ExpectedStatus
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 15
	}
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 3
	}
	if cfg.Dashboard.Port <= 0 {
		cfg.Dashboard.Port = 8080
	}
	if cfg.Storage.DBPath == "" {
		cfg.Storage.DBPath = "orbital-sentinel.db"
	}

	if len(cfg.Targets) == 0 {
		return nil, fmt.Errorf("no targets defined in config")
	}

	for i := range cfg.Targets {
		if cfg.Targets[i].Method == "" {
			cfg.Targets[i].Method = "GET"
		}
		if cfg.Targets[i].ExpectedStatus == 0 && len(cfg.Targets[i].AllowedStatuses) == 0 {
			cfg.Targets[i].ExpectedStatus = 200
		}
		if cfg.Targets[i].TimeoutSeconds <= 0 {
			cfg.Targets[i].TimeoutSeconds = 10
		}
	}

	return cfg, nil
}
