package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/agoodway/workie/config"
	"github.com/tmc/langchaingo/llms/ollama"
)

func init() {
	rootCmd.AddCommand(askCmd)
}

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask a question to the AI model based on the current configuration",
	Long:  `This command sends a question to the configured AI model and returns the response.`,
	Args:  cobra.ExactArgs(1),
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

		// Send question to model
		ctx := context.Background()
		response, err := llm.Call(ctx, question)
		if err != nil {
			fmt.Printf("Failed to get response: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("AI Response:", response)
	},
}
