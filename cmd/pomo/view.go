package main

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// segDigit is a 3-row × 3-char segment-display font for digits 0–9.
// Characters: space, underscore (_), pipe (|).
var segDigit = [10][3]string{
	{" _ ", "| |", "|_|"}, // 0
	{"   ", "  |", "  |"}, // 1
	{" _ ", " _|", "|_ "}, // 2
	{" _ ", " _|", " _|"}, // 3
	{"   ", "|_|", "  |"}, // 4
	{" _ ", "|_ ", " _|"}, // 5
	{" _ ", "|_ ", "|_|"}, // 6
	{" _ ", "  |", "  |"}, // 7
	{" _ ", "|_|", "|_|"}, // 8
	{" _ ", "|_|", " _|"}, // 9
}

var segColon = [3]string{" ", ":", " "} // 1-wide colon

// Colour palette (256-colour, works on all common terminals).
var (
	colDim    = lipgloss.Color("240")
	colBright = lipgloss.Color("253")
	colAccent = lipgloss.Color("75")  // steel blue
	colGreen  = lipgloss.Color("76")  // running
	colYellow = lipgloss.Color("220") // paused
	colOrange = lipgloss.Color("208") // completed dots
	colBg     = lipgloss.Color("237") // key cap background
)

var (
	dimSt    = lipgloss.NewStyle().Foreground(colDim)
	clockSt  = lipgloss.NewStyle().Foreground(colBright).Bold(true)
	keyCapSt = lipgloss.NewStyle().Background(colBg).Foreground(colBright).Padding(0, 1)
	badgeSt  = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colDim).
			Foreground(colAccent).
			Padding(0, 1)
	dotDoneSt = lipgloss.NewStyle().Foreground(colOrange)
	dotNowSt  = lipgloss.NewStyle().Foreground(colAccent)
	dotNextSt = lipgloss.NewStyle().Foreground(colDim)
	runSt     = lipgloss.NewStyle().Foreground(colGreen)
	pauseSt   = lipgloss.NewStyle().Foreground(colYellow)
	taskDimSt = lipgloss.NewStyle().Foreground(colDim).Italic(true)
)

// bigClockRows renders a "MM:SS" string as 3-row ASCII art.
func bigClockRows(timeStr string) [3]string {
	var rows [3]strings.Builder
	for i, ch := range timeStr {
		if i > 0 {
			for r := range rows {
				rows[r].WriteByte(' ')
			}
		}
		var seg [3]string
		switch {
		case ch >= '0' && ch <= '9':
			seg = segDigit[ch-'0']
		case ch == ':':
			seg = segColon
		default:
			seg = [3]string{"   ", "   ", "   "}
		}
		for r, line := range seg {
			rows[r].WriteString(line)
		}
	}
	return [3]string{rows[0].String(), rows[1].String(), rows[2].String()}
}

// viewBarWidth mirrors the WindowSizeMsg bar-width formula.
func viewBarWidth(termWidth int) int {
	const padding = 4
	const timeDisplayWidth = 20
	const minBarWidth = 20
	const maxBarWidth = 80
	w := termWidth - padding - timeDisplayWidth
	if w > maxBarWidth {
		w = maxBarWidth
	}
	if w < minBarWidth {
		w = minBarWidth
	}
	return w
}

// minuteMarkers renders a row of minute tick labels aligned to barWidth.
func minuteMarkers(elapsed, total time.Duration, barWidth int) string {
	totalMins := int(total.Minutes())
	if totalMins == 0 || barWidth <= 0 {
		return ""
	}

	step := 10
	if totalMins <= 15 {
		step = 5
	} else if totalMins > 90 {
		step = 20
	}

	// Build raw byte buffer; "+5" gives room for trailing "NNNm" label.
	buf := make([]byte, barWidth+5)
	for i := range buf {
		buf[i] = ' '
	}

	for t := 0; t <= totalMins; t += step {
		pos := t * barWidth / totalMins
		var label string
		if t == totalMins {
			label = fmt.Sprintf("%dm", t)
		} else {
			label = fmt.Sprintf("%d", t)
		}
		for j := 0; j < len(label) && pos+j < len(buf); j++ {
			buf[pos+j] = label[j]
		}
	}

	end := barWidth + 4
	if end > len(buf) {
		end = len(buf)
	}
	return dimSt.Render(string(buf[:end]))
}

// pomodoroDots renders the 4-dot cycle indicator and its label.
func pomodoroDots(m Model) string {
	const setSize = 4
	currentPos := m.intervalsCompleted % setSize

	// During rest, the just-completed interval counts as done.
	doneCount := currentPos
	if m.isRest {
		doneCount = currentPos + 1
	}

	var sb strings.Builder
	for i := 0; i < setSize; i++ {
		if i > 0 {
			sb.WriteByte(' ')
		}
		switch {
		case i < doneCount:
			sb.WriteString(dotDoneSt.Render("●"))
		case i == currentPos && !m.isRest:
			sb.WriteString(dotNowSt.Render("◉"))
		default:
			sb.WriteString(dotNextSt.Render("○"))
		}
	}

	labelPos := currentPos + 1
	label := fmt.Sprintf("  pomodoro %d / %d", labelPos, setSize)
	if !m.isRest && currentPos == setSize-1 {
		label += " · long break after next"
	}
	sb.WriteString(dimSt.Render(label))
	return sb.String()
}

type kbBinding struct{ key, desc string }

func keybindingGrid(m Model) [][2]kbBinding {
	if m.isRest || m.isLunch {
		return [][2]kbBinding{
			{{"space", "pause / resume"}, {"s", "skip interval"}},
			{{"q", "quit"}, {"", ""}},
		}
	}
	return [][2]kbBinding{
		{{"space", "pause / resume"}, {"n", "rename session"}},
		{{"s", "skip interval"}, {"l", "lunch break"}},
		{{"a", "add task"}, {"r", "reset"}},
		{{"q", "quit"}, {"", ""}},
	}
}

func renderKeybindings(m Model) string {
	grid := keybindingGrid(m)
	const colWidth = 34
	var sb strings.Builder
	for _, row := range grid {
		for col, b := range row {
			if b.key == "" {
				continue
			}
			cap := keyCapSt.Render(b.key)
			cell := cap + " " + dimSt.Render(b.desc)
			if col == 0 {
				cellW := lipgloss.Width(cap) + 1 + len(b.desc)
				if cellW < colWidth {
					cell += strings.Repeat(" ", colWidth-cellW)
				}
				sb.WriteString("  " + cell)
			} else {
				sb.WriteString("  " + cell)
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func renderStatusBar(m Model) string {
	var dot, state string
	if m.paused {
		dot = pauseSt.Render("●")
		state = "paused"
	} else {
		dot = runSt.Render("●")
		state = "running"
	}

	totalWorked := m.totalWorked
	if !m.isRest && !m.isLunch && m.remaining > 0 {
		if elapsed := m.workDuration - m.remaining; elapsed > 0 {
			totalWorked += elapsed
		}
	}

	sep := dimSt.Render(" · ")
	parts := []string{dot + " " + dimSt.Render(state)}
	if totalWorked > 0 {
		parts = append(parts, dimSt.Render(formatDurationHuman(totalWorked)+" focused"))
	}
	return "  " + strings.Join(parts, sep)
}
