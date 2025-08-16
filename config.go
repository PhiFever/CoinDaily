package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	CoinGecko struct {
		APIKey string `yaml:"api_key"`
	} `yaml:"coingecko"`
	
	Email struct {
		SMTPServer string `yaml:"smtp_server"`
		SMTPPort   int    `yaml:"smtp_port"`
		Username   string `yaml:"username"`
		Password   string `yaml:"password"`
		To         string `yaml:"to"`
	} `yaml:"email"`
	
	Coins []string `yaml:"coins"`
	
	Schedule struct {
		Hour   int `yaml:"hour"`
		Minute int `yaml:"minute"`
	} `yaml:"schedule"`
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

	return &config, nil
}

func validateConfig(config *Config) error {
	if config.CoinGecko.APIKey == "" {
		return fmt.Errorf("coingecko.api_key is required")
	}
	if config.Email.SMTPServer == "" {
		return fmt.Errorf("email.smtp_server is required")
	}
	if config.Email.SMTPPort == 0 {
		return fmt.Errorf("email.smtp_port is required")
	}
	if config.Email.Username == "" {
		return fmt.Errorf("email.username is required")
	}
	if config.Email.Password == "" {
		return fmt.Errorf("email.password is required")
	}
	if config.Email.To == "" {
		return fmt.Errorf("email.to is required")
	}
	if len(config.Coins) == 0 {
		return fmt.Errorf("at least one coin must be specified")
	}
	if config.Schedule.Hour < 0 || config.Schedule.Hour > 23 {
		return fmt.Errorf("schedule.hour must be between 0 and 23")
	}
	if config.Schedule.Minute < 0 || config.Schedule.Minute > 59 {
		return fmt.Errorf("schedule.minute must be between 0 and 59")
	}

	return nil
}