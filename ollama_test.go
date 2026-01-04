package main

import (
	"strings"
	"testing"
)

func TestCleanCommitMessage(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Simple commit message",
			input:    "feat: add new feature",
			expected: "feat: add new feature",
			wantErr:  false,
		},
		{
			name:     "Message with prefix",
			input:    "commit message: feat: add new feature",
			expected: "feat: add new feature",
			wantErr:  false,
		},
		{
			name:     "Message with Commit message: prefix",
			input:    "Commit message: feat: add new feature",
			expected: "feat: add new feature",
			wantErr:  false,
		},
		{
			name:     "Message in backticks",
			input:    "`feat: add new feature`",
			expected: "feat: add new feature",
			wantErr:  false,
		},
		{
			name:     "Message in quotes",
			input:    "\"feat: add new feature\"",
			expected: "feat: add new feature",
			wantErr:  false,
		},
		{
			name:     "Message with newlines",
			input:    "feat: add new feature\n\nThis is a description",
			expected: "feat: add new feature",
			wantErr:  false,
		},
		{
			name:     "Empty message",
			input:    "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Only prefix",
			input:    "commit message:",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Only whitespace",
			input:    "   \n\n   ",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Only backticks",
			input:    "```",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the cleanup logic from GenerateCommitMessage
			message := strings.TrimSpace(tc.input)

			if message == "" && tc.wantErr {
				return // Expected error
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

			result := firstLine

			// Check if we should have an error
			if tc.wantErr {
				if strings.TrimSpace(result) != "" {
					t.Errorf("Expected error for input %q, but got result %q", tc.input, result)
				}
				return
			}

			if result != tc.expected {
				t.Errorf("Input: %q\nExpected: %q\nGot: %q", tc.input, tc.expected, result)
			}
		})
	}
}
