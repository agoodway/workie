package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/agoodway/workie/config"
	"github.com/agoodway/workie/tools"
	"github.com/tmc/langchaingo/llms/ollama"
)

var (
	useTools bool
	askVerbose bool
)

func init() {
	askCmd.Flags().BoolVarP(&useTools, "tools", "t", false, "Enable tool/function calling for system commands")
	askCmd.Flags().BoolVarP(&askVerbose, "verbose", "v", false, "Show verbose output including tool calls")
	rootCmd.AddCommand(askCmd)
}

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask a question to the AI model based on the current configuration",
	Long: `This command sends a question to the configured AI model and returns the response.

With the --tools flag, the AI can execute system commands to answer questions like:
- "What is the current git branch?"
- "List files in the current directory"
- "Show the contents of README.md"
- "What is the current working directory?"
- "Create a commit message based on the files changed"
- "Generate a detailed commit message"`,
	Example: `  # Simple question without tools
  workie ask "What is Git?"
  
  # Question with tool execution
  workie ask --tools "What is the current branch name?"
  
  # Verbose mode to see tool calls
  workie ask --tools --verbose "Show me the last 5 git commits"
  
  # Generate commit message
  workie ask --tools "Create a commit message based on the files changed"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		question := args[0]

		// Load configuration using Viper
		config, err := config.LoadConfigWithViper("./", "config.yaml")
		if err != nil {
			fmt.Printf("Failed to load configuration: %v\n", err)
			os.Exit(1)
		}

		// Get model details
		modelProvider := config.AI.Model.Provider
		if modelProvider != "ollama" {
			fmt.Println("Only 'ollama' provider is supported for 'ask' command.")
			os.Exit(1)
		}

		// Create a LangChainGo Ollama client
		llm, err := ollama.New(
			ollama.WithModel(config.AI.Model.Name),
			ollama.WithServerURL(config.AI.Ollama.BaseURL),
		)
		if err != nil {
			fmt.Printf("Failed to create Ollama client: %v\n", err)
			os.Exit(1)
		}

		ctx := context.Background()

		if useTools {
			// Set up tool registry
			registry := tools.NewToolRegistry()
			registry.Register(tools.NewGitTool())
			registry.Register(tools.NewShellTool())
			registry.Register(tools.NewFileSystemTool())
			registry.Register(tools.NewCommitMessageTool())

			// Use SimpleAgent for better handling
			agent := tools.NewSimpleAgent(llm, registry, askVerbose)

			// Execute with tools
			response, err := agent.Execute(ctx, question)
			if err != nil {
				fmt.Printf("Failed to execute with tools: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(response)
		} else {
			// Direct LLM call without tools
			response, err := llm.Call(ctx, question)
			if err != nil {
				fmt.Printf("Failed to get response: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("AI Response:", response)
		}
	},
}
