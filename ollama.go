package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const ollamaURL = "http://localhost:11434"

type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options"`
}

type OllamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

// CheckOllamaRunning checks if Ollama is running
func CheckOllamaRunning() bool {
	resp, err := http.Get(ollamaURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GenerateCommitMessage generates a commit message using Ollama
func GenerateCommitMessage(diff string, seed int) (string, error) {
	prompt := fmt.Sprintf(`You are a git commit message generator. Analyze the diff and generate ONE commit message.

STRICT RULES:
1. Use conventional commits format: <type>: <description>
2. Valid types: feat, fix, docs, style, refactor, test, chore
3. Keep description under 72 characters
4. Be specific about what changed
5. Output ONLY the commit message - no prefixes like "commit message:" or "message:"
6. Do NOT include explanations, markdown, or additional text
7. Do NOT output empty text

EXAMPLES:
- feat: add user authentication system
- fix: resolve memory leak in cache
- docs: update installation guide

Git diff:
%s

Now output just the commit message:`, diff)

	reqBody := OllamaRequest{
		Model:  "phi4",
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.3,
			"seed":        seed,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama API error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", err
	}

	// Clean up the response
	message := strings.TrimSpace(ollamaResp.Response)

	// Handle empty response
	if message == "" {
		return "", fmt.Errorf("ollama returned empty response")
	}

	firstLine := strings.Split(message, "\n")[0]
	firstLine = strings.TrimSpace(firstLine)

	// Remove common prefixes (case-insensitive)
	prefixes := []string{
		"commit message:",
		"Commit message:",
		"COMMIT MESSAGE:",
		"message:",
		"Message:",
		"MESSAGE:",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(firstLine), strings.ToLower(prefix)) {
			firstLine = strings.TrimSpace(firstLine[len(prefix):])
			break
		}
	}

	// Remove markdown code blocks if present
	firstLine = strings.Trim(firstLine, "`")

	// Remove quotes if the entire message is quoted
	if (strings.HasPrefix(firstLine, "\"") && strings.HasSuffix(firstLine, "\"")) ||
		(strings.HasPrefix(firstLine, "'") && strings.HasSuffix(firstLine, "'")) {
		firstLine = strings.Trim(firstLine, "\"'")
		firstLine = strings.TrimSpace(firstLine)
	}

	// Final validation - ensure we have a non-empty message
	if strings.TrimSpace(firstLine) == "" {
		return "", fmt.Errorf("failed to extract commit message from AI response: %q", message)
	}

	return firstLine, nil
}
