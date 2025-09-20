// Package config handles DevDrop configuration management.
//
// This package manages the ~/.devdrop/config.yaml file which stores:
// - DockerHub username (from devdrop login)
// - Last container ID (from devdrop init)
// - Environment history and metadata
// - User preferences and settings
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Username      string                 `yaml:"username"`
	BaseImage     string                 `yaml:"base_image"`
	LastContainer string                 `yaml:"last_container,omitempty"`
	AuthToken     string                 `yaml:"auth_token,omitempty"`
	Environments  map[string]Environment `yaml:"environments"`
}

type Environment struct {
	Image       string    `yaml:"image"`
	Created     time.Time `yaml:"created"`
	LastUpdated time.Time `yaml:"last_updated"`
	Description string    `yaml:"description,omitempty"`
}

const (
	configDir  = ".devdrop"
	configFile = "config.yaml"
	defaultBaseImage = "ubuntu:24.04"
)

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, configDir, configFile), nil
}

// Load reads the configuration from ~/.devdrop/config.yaml
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{
			BaseImage:    defaultBaseImage,
			Environments: make(map[string]Environment),
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Ensure environments map is initialized
	if config.Environments == nil {
		config.Environments = make(map[string]Environment)
	}

	return &config, nil
}

// Save writes the configuration to ~/.devdrop/config.yaml
func (c *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SetUsername updates the username and saves the config
func (c *Config) SetUsername(username string) error {
	c.Username = username
	return c.Save()
}

// SetAuthToken updates the auth token and saves the config
func (c *Config) SetAuthToken(authToken string) error {
	c.AuthToken = authToken
	return c.Save()
}

// SetLastContainer updates the last container ID and saves the config
func (c *Config) SetLastContainer(containerID string) error {
	c.LastContainer = containerID
	return c.Save()
}

// GetPersonalImageName returns the user's personal image name
func (c *Config) GetPersonalImageName() string {
	if c.Username == "" {
		return ""
	}
	return fmt.Sprintf("%s/devdrop-env:latest", c.Username)
}

// AddEnvironment adds a new environment to the config
func (c *Config) AddEnvironment(name string, env Environment) error {
	c.Environments[name] = env
	return c.Save()
}