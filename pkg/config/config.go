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
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Username           string                 `yaml:"username"`
	BaseImage          string                 `yaml:"base_image"`
	LastContainer      string                 `yaml:"last_container,omitempty"`
	AuthToken          string                 `yaml:"auth_token,omitempty"`
	CurrentEnvironment string                 `yaml:"current_environment,omitempty"`
	Environments       map[string]Environment `yaml:"environments"`
}

type Environment struct {
	Image         string    `yaml:"image"`
	BaseImage     string    `yaml:"base_image"`
	Created       time.Time `yaml:"created"`
	LastUpdated   time.Time `yaml:"last_updated"`
	Description   string    `yaml:"description,omitempty"`
	LastContainer string    `yaml:"last_container,omitempty"`
}

const (
	configDir        = ".devdrop"
	configFile       = "config.yaml"
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

// EnsureDevDropPrefix ensures the environment name has the devdrop- prefix
func EnsureDevDropPrefix(envName string) string {
	if envName == "" {
		return "devdrop-default"
	}
	if !strings.HasPrefix(envName, "devdrop-") {
		return "devdrop-" + envName
	}
	return envName
}

// SetEnvironmentContainer updates the last container ID for a specific environment
func (c *Config) SetEnvironmentContainer(envName, containerID string) error {
	envName = EnsureDevDropPrefix(envName)
	env, exists := c.Environments[envName]
	if !exists {
		env = Environment{}
	}
	env.LastContainer = containerID
	env.LastUpdated = time.Now()
	c.Environments[envName] = env
	return c.Save()
}

// GetEnvironmentImageName returns the image name for a specific environment
func (c *Config) GetEnvironmentImageName(envName string) string {
	if c.Username == "" {
		return ""
	}
	envName = EnsureDevDropPrefix(envName)
	return fmt.Sprintf("%s/%s:latest", c.Username, envName)
}

// SetCurrentEnvironment sets the active environment
func (c *Config) SetCurrentEnvironment(envName string) error {
	envName = EnsureDevDropPrefix(envName)
	c.CurrentEnvironment = envName
	return c.Save()
}

// GetCurrentEnvironment returns the current environment, with fallback logic
func (c *Config) GetCurrentEnvironment() string {
	// If current environment is set and exists, use it
	if c.CurrentEnvironment != "" {
		if _, exists := c.Environments[c.CurrentEnvironment]; exists {
			return c.CurrentEnvironment
		}
	}

	// Fallback: use the most recently updated environment
	var latestEnv string
	var latestTime time.Time
	for name, env := range c.Environments {
		if env.LastUpdated.After(latestTime) {
			latestTime = env.LastUpdated
			latestEnv = name
		}
	}

	return latestEnv
}

// HasEnvironments returns true if any environments are configured
func (c *Config) HasEnvironments() bool {
	return len(c.Environments) > 0
}
