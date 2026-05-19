package main

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	colDim    = lipgloss.Color("240")
	colBright = lipgloss.Color("253")
)

var (
	dimSt     = lipgloss.NewStyle().Foreground(colDim)
	taskDimSt = lipgloss.NewStyle().Foreground(colDim).Italic(true)
	keySt     = lipgloss.NewStyle().Foreground(colBright)
)

// barWidth returns the progress-bar width for the given terminal width.
func barWidth(termWidth int) int {
	const minBarWidth = 20
	const maxBarWidth = 80
	const indent = 4
	w := termWidth - indent
	if w > maxBarWidth {
		w = maxBarWidth
	}
	if w < minBarWidth {
		w = minBarWidth
	}
	return w
}

// keyHints renders the bottom row of available keys.
func keyHints(isRest bool) string {
	var hints []string
	if isRest {
		hints = []string{
			keySt.Render("[space]") + dimSt.Render(" pause"),
			keySt.Render("[s]") + dimSt.Render(" skip"),
			keySt.Render("[q]") + dimSt.Render(" quit"),
		}
	} else {
		hints = []string{
			keySt.Render("[space]") + dimSt.Render(" pause"),
			keySt.Render("[a]") + dimSt.Render(" task"),
			keySt.Render("[s]") + dimSt.Render(" skip"),
			keySt.Render("[q]") + dimSt.Render(" quit"),
		}
	}
	return strings.Join(hints, "   ")
}
