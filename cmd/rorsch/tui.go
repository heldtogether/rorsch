package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/heldtogether/rorsch/internal"
)

func RenderTable(m *model) string {
	var s string

	bottomBorder := lipgloss.Border{
		Bottom: "─", // or "━" or whatever style you like
	}

	headerStyle := lipgloss.NewStyle().
		Border(bottomBorder).
		BorderBottom(true).
		PaddingLeft(0).
		PaddingRight(0)

	s += headerStyle.Render(
		fmt.Sprintf("  %-10s %-30s %-8s", "Name", "Command", "Status"),
	)
	s += "\n"

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color("#44475a")).
		Foreground(lipgloss.Color("#f8f8f2"))

	for i, command := range m.commands {

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		var status string
		switch command.Status {
		case internal.StatusTrying:
			status = "  " + m.spinner.View()
		case internal.StatusOk:
			status = "  ✅"
		case internal.StatusFailed:
			status = "  ❌"
		}

		row := fmt.Sprintf(
			"%s %-10s %-30s %-8s",
			cursor, command.Name, command.Exec, status,
		)

		if m.cursor == i {
			s += selectedStyle.Render(row) + "\n"
		} else {
			s += row + "\n"
		}
	}

	return s
}

func RenderMenu(m *model) string {
	var style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		PaddingTop(0).
		PaddingLeft(1).
		Width(m.width)

	return "\n" + style.Render(fmt.Sprintf("Press q to quit")) + "\n"
}

func tableHeight(m *model) int {
	return 2 + len(m.commands) // 2 for header row and border
}

func layoutOverhead(m *model) int {
	return 3 + // title
		tableHeight(m) +
		3 + // menu + surrounding line breaks
		2 // safety margin (e.g. spacing, padding)
}
