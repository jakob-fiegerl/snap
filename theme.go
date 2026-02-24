package main

import "github.com/charmbracelet/lipgloss"

// Snap color palette
var (
	colorPrimary   = lipgloss.Color("#53917E") // Sage teal — brand, accents, cursors
	colorSecondary = lipgloss.Color("#EF7B45") // Warm orange — warnings, hashes, unstaged
	colorSuccess   = lipgloss.Color("#6BCB77") // Green — success, staged, additions
	colorError     = lipgloss.Color("#E05263") // Coral red — errors, deletions
	colorMuted     = lipgloss.Color("#8B95A5") // Slate gray — info, dim text, borders
	colorText      = lipgloss.Color("#E5E7EB") // Soft white — default text
	colorBranch    = lipgloss.Color("#61AFEF") // Blue — main/master branch highlight
)
