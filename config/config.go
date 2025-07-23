package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// AIDecisionConfig represents AI-powered hook decision configuration
type AIDecisionConfig struct {
	Enabled    bool   `yaml:"enabled" mapstructure:"enabled"`                       // Enable AI decision making
	Model      string `yaml:"model,omitempty" mapstructure:"model"`                // Override model (uses default if empty)
	StrictMode bool   `yaml:"strict_mode,omitempty" mapstructure:"strict_mode"`     // If true, any hook failure = block
}

// Hooks represents the configuration for lifecycle hooks
type Hooks struct {
	PostCreate     []string `yaml:"post_create" mapstructure:"post_create"`
	PreRemove      []string `yaml:"pre_remove" mapstructure:"pre_remove"`
	TimeoutMinutes int      `yaml:"timeout_minutes,omitempty" mapstructure:"timeout_minutes"` // Hook execution timeout in minutes (default: 5)
	
	// Claude Code hook events
	ClaudePreToolUse       []string `yaml:"claude_pre_tool_use,omitempty" mapstructure:"claude_pre_tool_use"`             // Before Claude uses a tool
	ClaudePostToolUse      []string `yaml:"claude_post_tool_use,omitempty" mapstructure:"claude_post_tool_use"`           // After Claude uses a tool
	ClaudeNotification     []string `yaml:"claude_notification,omitempty" mapstructure:"claude_notification"`             // On Claude notifications
	ClaudeUserPromptSubmit []string `yaml:"claude_user_prompt_submit,omitempty" mapstructure:"claude_user_prompt_submit"` // When user submits prompt
	ClaudeStop             []string `yaml:"claude_stop,omitempty" mapstructure:"claude_stop"`                           // When Claude finishes responding
	ClaudeSubagentStop     []string `yaml:"claude_subagent_stop,omitempty" mapstructure:"claude_subagent_stop"`         // When subagent finishes
	ClaudePreCompact       []string `yaml:"claude_pre_compact,omitempty" mapstructure:"claude_pre_compact"`             // Before context compaction
	
	// AI decision configuration
	AIDecision *AIDecisionConfig `yaml:"ai_decision,omitempty" mapstructure:"ai_decision"`
}

// AIModel represents AI model configuration
type AIModel struct {
	Provider       string  `yaml:"provider" mapstructure:"provider"`
	Name           string  `yaml:"name" mapstructure:"name"`
	Version        string  `yaml:"version" mapstructure:"version"`
	Temperature    float64 `yaml:"temperature" mapstructure:"temperature"`
	MaxTokens      int     `yaml:"max_tokens" mapstructure:"max_tokens"`
	ContextLength  int     `yaml:"context_length" mapstructure:"context_length"`
	TopP           float64 `yaml:"top_p" mapstructure:"top_p"`
	Timeout        int     `yaml:"timeout" mapstructure:"timeout"`
}

// OllamaConfig represents Ollama-specific configuration
type OllamaConfig struct {
	BaseURL   string            `yaml:"base_url" mapstructure:"base_url"`
	Endpoints map[string]string `yaml:"endpoints" mapstructure:"endpoints"`
	KeepAlive string            `yaml:"keep_alive" mapstructure:"keep_alive"`
	NumThread int               `yaml:"num_thread" mapstructure:"num_thread"`
	NumGPU    int               `yaml:"num_gpu" mapstructure:"num_gpu"`
}

// AIConfig represents AI configuration
type AIConfig struct {
	Enabled bool          `yaml:"enabled" mapstructure:"enabled"`
	Model   AIModel       `yaml:"model" mapstructure:"model"`
	Ollama  OllamaConfig  `yaml:"ollama" mapstructure:"ollama"`
}

// Config represents the YAML configuration structure
type Config struct {
	FilesToCopy     []string               `yaml:"files_to_copy" mapstructure:"files_to_copy"`
	Hooks           *Hooks                 `yaml:"hooks,omitempty" mapstructure:"hooks"`
	AI              AIConfig               `yaml:"ai" mapstructure:"ai"`
	Providers       map[string]interface{} `yaml:"providers,omitempty" mapstructure:"providers"`       // Provider configurations
	DefaultProvider string                 `yaml:"default_provider,omitempty" mapstructure:"default_provider"` // Default issue provider
	LoadedFrom      string                 `yaml:"-" mapstructure:"-"` // Path to the loaded config file (not serialized)
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
		// Try default config file names in the repository root
		configNames := []string{".workie.yaml", "workie.yaml"}
		for _, name := range configNames {
			path := filepath.Join(repoPath, name)
			if _, err := os.Stat(path); err == nil {
				configPath = path
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
	
	// Set defaults
	v.SetDefault("ai.model.provider", "ollama")
	v.SetDefault("ai.model.name", "llama3.2")
	v.SetDefault("ai.model.temperature", 0.7)
	v.SetDefault("ai.model.max_tokens", 2048)
	v.SetDefault("ai.model.context_length", 4096)
	v.SetDefault("ai.model.top_p", 0.9)
	v.SetDefault("ai.model.timeout", 60)
	v.SetDefault("ai.ollama.base_url", "http://localhost:11434")
	v.SetDefault("ai.ollama.keep_alive", "5m")
	v.SetDefault("ai.ollama.num_thread", 4)
	v.SetDefault("ai.ollama.num_gpu", 0)
	
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
		v.AddConfigPath(repoRoot)        // Repository root (highest priority)
		v.AddConfigPath(".")            // Current directory
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

// IsAIEnabled returns true if AI features are enabled
func (c *Config) IsAIEnabled() bool {
	return c != nil && c.AI.Model.Provider != "" && c.AI.Model.Name != ""
}

// GetOllamaEndpoint returns the full Ollama API endpoint for a given operation
func (c *Config) GetOllamaEndpoint(operation string) string {
	if c.AI.Ollama.Endpoints != nil {
		if endpoint, ok := c.AI.Ollama.Endpoints[operation]; ok {
			return fmt.Sprintf("%s%s", c.AI.Ollama.BaseURL, endpoint)
		}
	}
	
	// Default endpoints
	defaults := map[string]string{
		"chat":     "/api/chat",
		"generate": "/api/generate",
		"tags":     "/api/tags",
		"pull":     "/api/pull",
	}
	
	if endpoint, ok := defaults[operation]; ok {
		return fmt.Sprintf("%s%s", c.AI.Ollama.BaseURL, endpoint)
	}
	
	return c.AI.Ollama.BaseURL
}

// Providers represents the issue provider configurations
type Providers struct {
	GitHub *GitHubProvider `yaml:"github,omitempty" mapstructure:"github"`
	Jira   *JiraProvider   `yaml:"jira,omitempty" mapstructure:"jira"`
	Linear *LinearProvider `yaml:"linear,omitempty" mapstructure:"linear"`
}

// GitHubProvider represents GitHub configuration
type GitHubProvider struct {
	Enabled      bool              `yaml:"enabled" mapstructure:"enabled"`
	Settings     GitHubSettings    `yaml:"settings" mapstructure:"settings"`
	BranchPrefix map[string]string `yaml:"branch_prefix,omitempty" mapstructure:"branch_prefix"`
}

// GitHubSettings contains GitHub-specific settings
type GitHubSettings struct {
	TokenEnv string `yaml:"token_env" mapstructure:"token_env"`
	Owner    string `yaml:"owner" mapstructure:"owner"`
	Repo     string `yaml:"repo" mapstructure:"repo"`
}

// JiraProvider represents Jira configuration
type JiraProvider struct {
	Enabled      bool              `yaml:"enabled" mapstructure:"enabled"`
	Settings     JiraSettings      `yaml:"settings" mapstructure:"settings"`
	BranchPrefix map[string]string `yaml:"branch_prefix,omitempty" mapstructure:"branch_prefix"`
}

// JiraSettings contains Jira-specific settings
type JiraSettings struct {
	BaseURL      string `yaml:"base_url" mapstructure:"base_url"`
	EmailEnv     string `yaml:"email_env" mapstructure:"email_env"`
	APITokenEnv  string `yaml:"api_token_env" mapstructure:"api_token_env"`
	Project      string `yaml:"project" mapstructure:"project"`
}

// LinearProvider represents Linear configuration
type LinearProvider struct {
	Enabled      bool              `yaml:"enabled" mapstructure:"enabled"`
	Settings     LinearSettings    `yaml:"settings" mapstructure:"settings"`
	BranchPrefix map[string]string `yaml:"branch_prefix,omitempty" mapstructure:"branch_prefix"`
}

// LinearSettings contains Linear-specific settings
type LinearSettings struct {
	APIKeyEnv string `yaml:"api_key_env" mapstructure:"api_key_env"`
	TeamID    string `yaml:"team_id,omitempty" mapstructure:"team_id"`
}