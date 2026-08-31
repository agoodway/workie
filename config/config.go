package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Hooks represents the configuration for lifecycle hooks
type Hooks struct {
	PostCreate     []string `yaml:"post_create" mapstructure:"post_create"`
	PreRemove      []string `yaml:"pre_remove" mapstructure:"pre_remove"`
	TimeoutMinutes int      `yaml:"timeout_minutes,omitempty" mapstructure:"timeout_minutes"` // Hook execution timeout in minutes (default: 5)
}

// SpriteConfig represents configuration for a sprite VM
type SpriteConfig struct {
	Name      string   `yaml:"name" mapstructure:"name"`
	Org       string   `yaml:"org,omitempty" mapstructure:"org"`
	PreSetup  []string `yaml:"pre_setup,omitempty" mapstructure:"pre_setup"`
	PostSetup []string `yaml:"post_setup,omitempty" mapstructure:"post_setup"`
}

// Config represents the YAML configuration structure
type Config struct {
	FilesToCopy []string      `yaml:"files_to_copy" mapstructure:"files_to_copy"`
	Hooks       *Hooks        `yaml:"hooks,omitempty" mapstructure:"hooks"`
	Sprite      *SpriteConfig `yaml:"sprite,omitempty" mapstructure:"sprite"`
	LoadedFrom  string        `yaml:"-" mapstructure:"-"` // Path to the loaded config file (not serialized)
}

// LoadConfig attempts to load configuration from the specified file path,
// falling back to default locations if no custom path is provided
func LoadConfig(repoPath, customPath string) (*Config, error) {
	config := &Config{}

	var configPath string

	if customPath != "" {
		// Use custom config file if specified
		configPath = customPath
		if !strings.HasPrefix(configPath, "/") {
			// Make relative paths relative to the current directory, not repo root
			cwd, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("failed to get current directory: %w", err)
			}
			configPath = filepath.Join(cwd, configPath)
		}

		// Check if custom config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("custom config file not found: %s", configPath)
		}
	} else {
		// Try default config file names in the repository root and current directory
		configNames := []string{".workie.yaml", "workie.yaml"}
		cwd, _ := os.Getwd()
		searchPaths := []string{repoPath}
		if cwd != "" && cwd != repoPath {
			searchPaths = append(searchPaths, cwd)
		}
		for _, dir := range searchPaths {
			for _, name := range configNames {
				path := filepath.Join(dir, name)
				if _, err := os.Stat(path); err == nil {
					configPath = path
					break
				}
			}
			if configPath != "" {
				break
			}
		}
	}

	// If no config file is found, return empty config (not an error)
	if configPath == "" {
		return config, nil
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Use yaml.v3 for parsing
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML from %s: %w", configPath, err)
	}

	// Set the path where config was loaded from
	config.LoadedFrom = configPath

	return config, nil
}

// HasFilesToCopy returns true if there are files to copy
func (c *Config) HasFilesToCopy() bool {
	return c != nil && len(c.FilesToCopy) > 0
}

// LoadConfigWithViper loads configuration using Viper library
// This provides enhanced features like environment variable support, defaults, etc.
func LoadConfigWithViper(repoRoot string, customConfigPath string) (*Config, error) {
	// Create a new Viper instance
	v := viper.New()

	// Environment variable support
	v.SetEnvPrefix("WORKIE")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Validate repository root
	if repoRoot == "" {
		return nil, fmt.Errorf("repository root path cannot be empty")
	}

	// Verify repo root exists and is accessible
	if info, err := os.Stat(repoRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("repository root does not exist: %s", repoRoot)
		}
		return nil, fmt.Errorf("cannot access repository root: %w", err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("repository root is not a directory: %s", repoRoot)
	}

	// If custom config path is provided, use it directly
	if customConfigPath != "" {
		v.SetConfigFile(customConfigPath)
	} else {
		// Set config search paths
		v.SetConfigName(".workie")
		v.SetConfigType("yaml")

		// Add search paths in priority order
		v.AddConfigPath(repoRoot)               // Repository root (highest priority)
		v.AddConfigPath(".")                    // Current directory
		v.AddConfigPath("$HOME/.config/workie") // User config directory

		// Also check for workie.yaml (without leading dot)
		// Note: Viper will check both .workie.yaml and workie.yaml
		v.SetConfigName("workie")
		v.AddConfigPath(repoRoot)
		v.AddConfigPath(".")
	}

	// Try to read the config file
	config := &Config{}

	if err := v.ReadInConfig(); err != nil {
		// If it's just a missing config file, use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// No config file found, populate with defaults
			if err := v.Unmarshal(config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal default config: %w", err)
			}
			return config, nil
		}

		// For actual errors (parse errors, permission issues, etc.)
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal configuration
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Store the loaded config file path
	config.LoadedFrom = v.ConfigFileUsed()

	return config, nil
}

