package tools

import (
	"context"
	"encoding/json"
)

// Tool represents a function that can be called by the AI
type Tool interface {
	// Name returns the name of the tool
	Name() string
	// Description returns a description of what the tool does
	Description() string
	// Parameters returns the JSON schema for the tool's parameters
	Parameters() map[string]interface{}
	// Execute runs the tool with the given parameters
	Execute(ctx context.Context, params map[string]interface{}) (string, error)
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools
func (r *ToolRegistry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ToolCall represents a request to execute a tool
type ToolCall struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolResponse represents the result of a tool execution
type ToolResponse struct {
	Name   string `json:"name"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

// FormatToolsPrompt creates a prompt that describes available tools
func FormatToolsPrompt(tools []Tool) string {
	toolDescriptions := "You are an AI assistant with access to tools that can execute system commands. You have access to the following tools:\n\n"
	
	for _, tool := range tools {
		params, _ := json.MarshalIndent(tool.Parameters(), "", "  ")
		toolDescriptions += "Tool: " + tool.Name() + "\n"
		toolDescriptions += "Description: " + tool.Description() + "\n"
		toolDescriptions += "Parameters: " + string(params) + "\n\n"
	}
	
	toolDescriptions += `When you need to use a tool to answer a question, respond with ONLY a JSON object in the following format:
{
  "tool": "tool_name",
  "parameters": {
    "param1": "value1",
    "param2": "value2"
  }
}

For example, to get the current git branch:
{
  "tool": "git",
  "parameters": {
    "command": "branch"
  }
}

Important: Only output the JSON when using a tool. Do not include any other text with the JSON.`
	
	return toolDescriptions
}

// ParseToolCall extracts a tool call from AI response
func ParseToolCall(response string) (*ToolCall, error) {
	// Try to find JSON in the response
	start := -1
	end := -1
	braceCount := 0
	
	for i, char := range response {
		if char == '{' {
			if start == -1 {
				start = i
			}
			braceCount++
		} else if char == '}' {
			braceCount--
			if braceCount == 0 && start != -1 {
				end = i + 1
				break
			}
		}
	}
	
	if start == -1 || end == -1 {
		return nil, nil // No JSON found
	}
	
	jsonStr := response[start:end]
	
	var rawCall map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawCall); err != nil {
		return nil, err
	}
	
	toolName, ok := rawCall["tool"].(string)
	if !ok {
		return nil, nil
	}
	
	params, ok := rawCall["parameters"].(map[string]interface{})
	if !ok {
		params = make(map[string]interface{})
	}
	
	return &ToolCall{
		Name:       toolName,
		Parameters: params,
	}, nil
}