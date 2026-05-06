package main

import (
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// activeTask returns a pointer to the currently active (not yet ended) task, or nil.
func (m *Model) activeTask() *Task {
	for i := len(m.tasks) - 1; i >= 0; i-- {
		if m.tasks[i].EndedAt.IsZero() {
			return &m.tasks[i]
		}
	}
	return nil
}

// endActiveTask closes the active task with the given end time.
func (m *Model) endActiveTask(at time.Time) {
	for i := len(m.tasks) - 1; i >= 0; i-- {
		if m.tasks[i].EndedAt.IsZero() {
			m.tasks[i].EndedAt = at
			return
		}
	}
}

// startTask ends any active task and starts a new one.
func (m *Model) startTask(name string, at time.Time) {
	m.endActiveTask(at)
	m.tasks = append(m.tasks, Task{Name: name, StartedAt: at})
}

// recentCategoryForTask returns the category of the most recent task entry
// matching name. The bool indicates whether a non-empty category was found.
func recentCategoryForTask(tasks []Task, name string) (string, bool) {
	for i := len(tasks) - 1; i >= 0; i-- {
		if tasks[i].Name == name && tasks[i].Category != "" {
			return tasks[i].Category, true
		}
	}
	return "", false
}

// recentTaskNames returns up to 9 unique task names from the session history,
// most-recently-started first, excluding the currently active task.
func recentTaskNames(tasks []Task) []string {
	activeName := ""
	for i := len(tasks) - 1; i >= 0; i-- {
		if tasks[i].EndedAt.IsZero() {
			activeName = tasks[i].Name
			break
		}
	}
	seen := make(map[string]bool)
	var names []string
	for i := len(tasks) - 1; i >= 0; i-- {
		name := tasks[i].Name
		if name == "" || name == activeName || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
		if len(names) == 9 {
			break
		}
	}
	return names
}

// handleNamingInput processes keyboard input while in naming mode (pomodoro rename).
func (m Model) handleNamingInput(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEnter:
		m.currentPomodoroName = m.nameInput
		m.namingMode = false
		m.nameInput = ""
	case tea.KeyEscape:
		m.namingMode = false
		m.nameInput = ""
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.nameInput) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.nameInput)
			m.nameInput = m.nameInput[:len(m.nameInput)-size]
		}
	default:
		if msg.Text != "" {
			m.nameInput += msg.Text
		}
	}
	return m, nil
}

// handleTaskInput processes keyboard input while in task mode.
func (m Model) handleTaskInput(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.taskInput)
		m.taskMode = false
		m.taskInput = ""
		m.recentTasks = nil
		if name != "" {
			if len(m.categories) > 0 {
				m.pendingTask = name
				m.categoryMode = true
			} else {
				m.startTask(name, time.Now())
			}
		}
	case tea.KeyEscape:
		m.taskMode = false
		m.taskInput = ""
		m.recentTasks = nil
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.taskInput) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.taskInput)
			m.taskInput = m.taskInput[:len(m.taskInput)-size]
		}
	default:
		// When the input is still empty and a digit is pressed, check the recent task list.
		if msg.Text != "" && m.taskInput == "" && len(m.recentTasks) > 0 {
			if d := msg.Text[0]; d >= '1' && d <= '9' {
				idx := int(d-'0') - 1
				if idx < len(m.recentTasks) {
					name := m.recentTasks[idx]
					m.taskMode = false
					m.recentTasks = nil
					if cat, found := recentCategoryForTask(m.tasks, name); found {
						m.startTask(name, time.Now())
						m.tasks[len(m.tasks)-1].Category = cat
					} else if len(m.categories) > 0 {
						m.pendingTask = name
						m.categoryMode = true
					} else {
						m.startTask(name, time.Now())
					}
					return m, nil
				}
			}
		}
		if msg.Text != "" {
			m.taskInput += msg.Text
		}
	}
	return m, nil
}

// handleCategoryInput processes keyboard input while in category mode (step 2 of task creation).
func (m Model) handleCategoryInput(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEscape:
		m.categoryMode = false
		m.pendingTask = ""
	default:
		if msg.Text == "0" {
			m.startTask(m.pendingTask, time.Now())
			m.categoryMode = false
			m.pendingTask = ""
		} else if len(msg.Text) == 1 && msg.Text[0] >= '1' && msg.Text[0] <= '9' {
			idx := int(msg.Text[0]-'0') - 1
			if idx < len(m.categories) {
				m.startTask(m.pendingTask, time.Now())
				m.tasks[len(m.tasks)-1].Category = m.categories[idx]
				m.categoryMode = false
				m.pendingTask = ""
			}
		}
	}
	return m, nil
}
