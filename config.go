package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	CoinGecko struct {
		APIKey string `yaml:"api_key"`
	} `yaml:"coingecko"`

	Alpaca struct {
		APIKey    string `yaml:"api_key"`
		SecretKey string `yaml:"secret_key"`
		Feed      string `yaml:"feed"`
	} `yaml:"alpaca"`

	Hyperliquid struct {
		Perpetuals []string `yaml:"perpetuals"`
	} `yaml:"hyperliquid"`

	Email struct {
		SMTPServer string   `yaml:"smtp_server"`
		SMTPPort   int      `yaml:"smtp_port"`
		Username   string   `yaml:"username"`
		Password   string   `yaml:"password"`
		To         []string `yaml:"to"`
	} `yaml:"email"`

	// Discord 配置（可选）
	Discord struct {
		BotToken  string `yaml:"bot_token"`
		ChannelID string `yaml:"channel_id"`
	} `yaml:"discord"`

	Proxy struct {
		Enabled bool   `yaml:"enabled"`
		URL     string `yaml:"url"`
	} `yaml:"proxy"`

	Coins  []string `yaml:"coins"`
	Stocks []string `yaml:"stocks"`

	Schedule struct {
		Hour   int `yaml:"hour"`
		Minute int `yaml:"minute"`
	} `yaml:"schedule"`

	configPath string
}

func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config file path: %w", err)
	}
	config.configPath = absPath

	return &config, nil
}

func validateConfig(config *Config) error {
	if len(config.Coins) > 0 && strings.TrimSpace(config.CoinGecko.APIKey) == "" {
		return fmt.Errorf("coingecko.api_key is required")
	}

	if config.Alpaca.Feed == "" {
		config.Alpaca.Feed = "delayed_sip"
	}
	if config.Alpaca.Feed != "delayed_sip" {
		return fmt.Errorf("alpaca.feed must be delayed_sip")
	}
	if len(config.Stocks) > 0 {
		if strings.TrimSpace(config.Alpaca.APIKey) == "" {
			return fmt.Errorf("alpaca.api_key is required when stocks are configured")
		}
		if strings.TrimSpace(config.Alpaca.SecretKey) == "" {
			return fmt.Errorf("alpaca.secret_key is required when stocks are configured")
		}
	}
	for _, symbol := range config.Hyperliquid.Perpetuals {
		parts := strings.Split(symbol, ":")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return fmt.Errorf("hyperliquid perpetual %q must include a DEX prefix such as xyz:ZHIPU", symbol)
		}
	}

	// 检查是否有至少一个通知渠道配置
	hasEmail := isEmailConfigured(config)
	hasDiscord := isDiscordConfigured(config)

	if !hasEmail && !hasDiscord {
		return fmt.Errorf("至少需要配置一个通知渠道 (email 或 discord)")
	}

	// 如果配置了邮件，验证邮件配置完整性
	if hasEmail {
		if config.Email.SMTPPort == 0 {
			return fmt.Errorf("email.smtp_port is required")
		}
		if config.Email.Username == "" {
			return fmt.Errorf("email.username is required")
		}
		if config.Email.Password == "" {
			return fmt.Errorf("email.password is required")
		}
		if len(config.Email.To) == 0 {
			return fmt.Errorf("email.to is required (at least one recipient)")
		}
	}

	// 如果配置了 Discord，验证 Discord 配置完整性
	if config.Discord.BotToken != "" || config.Discord.ChannelID != "" {
		if config.Discord.BotToken == "" {
			return fmt.Errorf("discord.bot_token is required when discord is configured")
		}
		if config.Discord.ChannelID == "" {
			return fmt.Errorf("discord.channel_id is required when discord is configured")
		}
	}

	if len(config.Coins) == 0 && len(config.Stocks) == 0 && len(config.Hyperliquid.Perpetuals) == 0 {
		return fmt.Errorf("at least one coin, stock, or perpetual must be specified")
	}
	if config.Schedule.Hour < 0 || config.Schedule.Hour > 23 {
		return fmt.Errorf("schedule.hour must be between 0 and 23")
	}
	if config.Schedule.Minute < 0 || config.Schedule.Minute > 59 {
		return fmt.Errorf("schedule.minute must be between 0 and 59")
	}

	return nil
}

func (c *Config) statePath() string {
	dir := "."
	if c.configPath != "" {
		dir = filepath.Dir(c.configPath)
	}
	return filepath.Join(dir, ".coindaily-state.json")
}

// isEmailConfigured 检查邮件配置是否存在
func isEmailConfigured(config *Config) bool {
	return config.Email.SMTPServer != ""
}

// isDiscordConfigured 检查 Discord 配置是否完整
func isDiscordConfigured(config *Config) bool {
	return config.Discord.BotToken != "" && config.Discord.ChannelID != ""
}
