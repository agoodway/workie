package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// OllamaAgent manages tool execution with Ollama
type OllamaAgent struct {
	llm      llms.Model
	registry *ToolRegistry
	verbose  bool
}

// NewOllamaAgent creates a new Ollama agent
func NewOllamaAgent(llm llms.Model, registry *ToolRegistry, verbose bool) *OllamaAgent {
	return &OllamaAgent{
		llm:      llm,
		registry: registry,
		verbose:  verbose,
	}
}

// Execute processes a user query and executes tools as needed
func (a *OllamaAgent) Execute(ctx context.Context, query string) (string, error) {
	// Build the system prompt with tool descriptions
	tools := a.registry.List()
	systemPrompt := FormatToolsPrompt(tools)
	
	// Combine system prompt with user query
	fullPrompt := systemPrompt + "\n\nUser Query: " + query + "\n\nThink about whether you need to use a tool to answer this query. If yes, respond with the appropriate tool JSON. If no, respond with the answer directly.\n\nAssistant:"
	
	// Keep track of conversation for multi-turn interactions
	conversation := []string{fullPrompt}
	maxIterations := 5
	
	for i := 0; i < maxIterations; i++ {
		// Get response from LLM
		response, err := a.llm.Call(ctx, strings.Join(conversation, "\n"))
		if err != nil {
			return "", fmt.Errorf("LLM call failed: %v", err)
		}
		
		if a.verbose {
			fmt.Printf("Iteration %d - LLM Response: %s\n", i+1, response)
		}
		
		// Check if the response contains a tool call
		toolCall, err := ParseToolCall(response)
		if err != nil {
			return "", fmt.Errorf("failed to parse tool call: %v", err)
		}
		
		// If no tool call found, return the response
		if toolCall == nil {
			return response, nil
		}
		
		// Execute the tool
		tool, exists := a.registry.Get(toolCall.Name)
		if !exists {
			errMsg := fmt.Sprintf("Tool '%s' not found", toolCall.Name)
			conversation = append(conversation, response)
			conversation = append(conversation, "Tool Error: " + errMsg)
			continue
		}
		
		if a.verbose {
			fmt.Printf("Executing tool: %s with parameters: %v\n", toolCall.Name, toolCall.Parameters)
		}
		
		result, err := tool.Execute(ctx, toolCall.Parameters)
		if err != nil {
			errMsg := fmt.Sprintf("Tool execution failed: %v", err)
			conversation = append(conversation, response)
			conversation = append(conversation, "Tool Error: " + errMsg)
			continue
		}
		
		// Add tool result to conversation
		conversation = append(conversation, response)
		conversation = append(conversation, fmt.Sprintf("Tool Result: %s", result))
		conversation = append(conversation, "Based on the tool result above, please provide a natural language answer to the user's original query. Be concise and direct.")
	}
	
	return "Maximum iterations reached. Unable to complete the request.", nil
}

// ExecuteWithHistory processes a query with conversation history
func (a *OllamaAgent) ExecuteWithHistory(ctx context.Context, query string, history []string) (string, error) {
	// Build the system prompt with tool descriptions
	tools := a.registry.List()
	systemPrompt := FormatToolsPrompt(tools)
	
	// Build conversation with history
	conversation := []string{systemPrompt}
	conversation = append(conversation, history...)
	conversation = append(conversation, "User: " + query)
	conversation = append(conversation, "Assistant:")
	
	fullPrompt := strings.Join(conversation, "\n")
	
	// Get response from LLM
	response, err := a.llm.Call(ctx, fullPrompt)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %v", err)
	}
	
	// Check if the response contains a tool call
	toolCall, err := ParseToolCall(response)
	if err != nil {
		return "", fmt.Errorf("failed to parse tool call: %v", err)
	}
	
	// If no tool call found, return the response
	if toolCall == nil {
		return response, nil
	}
	
	// Execute the tool
	tool, exists := a.registry.Get(toolCall.Name)
	if !exists {
		return fmt.Sprintf("I tried to use tool '%s' but it's not available. %s", toolCall.Name, response), nil
	}
	
	if a.verbose {
		fmt.Printf("Executing tool: %s with parameters: %v\n", toolCall.Name, toolCall.Parameters)
	}
	
	result, err := tool.Execute(ctx, toolCall.Parameters)
	if err != nil {
		return fmt.Sprintf("Tool execution failed: %v\n\nOriginal response: %s", err, response), nil
	}
	
	// Get final response based on tool result
	finalPrompt := strings.Join(conversation, "\n") + "\n" + response + 
		"\nTool Result: " + result + 
		"\n\nNow provide a clear, natural language answer to the user's query based on the tool result above. For example, if asked 'what is the current branch?' and the tool returned 'main', say 'The current branch is main.'"
	
	finalResponse, err := a.llm.Call(ctx, finalPrompt)
	if err != nil {
		// Fallback to formatting the tool result nicely
		if toolCall.Name == "git" && toolCall.Parameters["command"] == "branch" {
			return fmt.Sprintf("The current branch is: %s", result), nil
		}
		return fmt.Sprintf("Tool result: %s", result), nil
	}
	
	return finalResponse, nil
}