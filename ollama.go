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
	prompt := fmt.Sprintf(`Generate a git commit message for the following diff. 

Rules:
- Use conventional commits format: <type>: <description>
- Types: feat, fix, docs, style, refactor, test, chore
- Keep description under 72 characters
- Be specific about what changed
- Output ONLY the commit message on the first line
- Do NOT include explanations, reasoning, or additional text

Git diff:
%s

Commit message (one line only):`, diff)

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
	firstLine := strings.Split(message, "\n")[0]
	firstLine = strings.TrimSpace(firstLine)

	// Remove common prefixes
	firstLine = strings.TrimPrefix(firstLine, "commit message:")
	firstLine = strings.TrimPrefix(firstLine, "Commit message:")
	firstLine = strings.TrimPrefix(firstLine, "message:")
	firstLine = strings.TrimPrefix(firstLine, "Message:")
	firstLine = strings.TrimSpace(firstLine)

	// Remove markdown code blocks if present
	firstLine = strings.TrimPrefix(firstLine, "`")
	firstLine = strings.TrimSuffix(firstLine, "`")

	return firstLine, nil
}
