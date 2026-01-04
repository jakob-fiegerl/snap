package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
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
	var input string

	if len(diff) <= 2000 {
		input = diff
	} else {
		// Chunk and summarize
		chunks := splitDiffIntoChunks(diff)
		if len(chunks) == 0 {
			return "", fmt.Errorf("no diff chunks to process")
		}

		var summaries []string
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, chunk := range chunks {
			wg.Add(1)
			go func(c string) {
				defer wg.Done()
				summary, err := SummarizeDiffChunk(c, seed)
				if err == nil {
					mu.Lock()
					summaries = append(summaries, summary)
					mu.Unlock()
				}
			}(chunk)
		}

		wg.Wait()

		if len(summaries) == 0 {
			return "", fmt.Errorf("failed to summarize any diff chunks")
		}

		input = strings.Join(summaries, "; ")
	}

	prompt := fmt.Sprintf(`You are a git commit message generator. Generate a SINGLE LINE conventional commit message.

CRITICAL REQUIREMENTS:
- Output EXACTLY ONE LINE ONLY
- Format: <type>: <description>
- Types: feat, fix, docs, style, refactor, test, chore
- Description under 72 characters
- NO explanations, NO markdown, NO extra text
- NO line breaks, NO paragraphs

WRONG OUTPUTS:
- "feat: add new feature\nThis adds..."
- "To modify the code so that it outputs..."
- "commit message: feat: add feature"

CORRECT OUTPUTS:
- "feat: add user authentication system"
- "fix: resolve memory leak in cache"
- "docs: update installation guide"

Changes:
%s

OUTPUT ONLY ONE LINE:`, input)

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

	// Take ONLY the first line - be very aggressive about this
	lines := strings.Split(message, "\n")
	firstLine := strings.TrimSpace(lines[0])

	// If first line is empty, try the second line
	if firstLine == "" && len(lines) > 1 {
		firstLine = strings.TrimSpace(lines[1])
	}

	// Remove common prefixes (case-insensitive)
	prefixes := []string{
		"commit message:",
		"Commit message:",
		"COMMIT MESSAGE:",
		"message:",
		"Message:",
		"MESSAGE:",
		"output:",
		"Output:",
		"OUTPUT:",
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

	// Remove any remaining line breaks or extra whitespace
	firstLine = strings.ReplaceAll(firstLine, "\n", " ")
	firstLine = strings.ReplaceAll(firstLine, "\r", " ")
	firstLine = strings.Join(strings.Fields(firstLine), " ")

	// Final validation - ensure we have a non-empty message
	if strings.TrimSpace(firstLine) == "" {
		return "", fmt.Errorf("failed to extract commit message from AI response: %q", message)
	}

	return firstLine, nil
}

// SummarizeDiffChunk summarizes a chunk of git diff
func SummarizeDiffChunk(chunk string, seed int) (string, error) {
	prompt := fmt.Sprintf(`Summarize the changes in this git diff chunk in a few words, focusing on what was added, modified, or removed.

Git diff chunk:
%s

Summary:`, chunk)

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

	message := strings.TrimSpace(ollamaResp.Response)
	if message == "" {
		return "", fmt.Errorf("ollama returned empty response")
	}

	// Take first line
	lines := strings.Split(message, "\n")
	firstLine := strings.TrimSpace(lines[0])

	// Clean up
	firstLine = strings.ReplaceAll(firstLine, "\n", " ")
	firstLine = strings.ReplaceAll(firstLine, "\r", " ")
	firstLine = strings.Join(strings.Fields(firstLine), " ")

	return firstLine, nil
}

// splitDiffIntoChunks splits a git diff into chunks by file
func splitDiffIntoChunks(diff string) []string {
	// Split on "diff --git" but keep the separator
	parts := strings.Split(diff, "\ndiff --git ")
	var chunks []string

	if len(parts) == 0 {
		return chunks
	}

	// First part might be empty or not start with diff --git
	if strings.TrimSpace(parts[0]) != "" {
		chunks = append(chunks, parts[0])
	}

	for i := 1; i < len(parts); i++ {
		chunks = append(chunks, "diff --git "+parts[i])
	}

	return chunks
}
