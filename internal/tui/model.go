package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"fmt"
	"strings"
)

//go:generate go tool minimock -i Runner -g
type Runner interface {
	Run(context.Context, func(string)) error
}
type event struct {
	text string
	done bool
	err  error
}
type Model struct {
	runner        Runner
	events        chan event
	cancel        context.CancelFunc
	lines         []string
	running, done bool
	err           error
}

func New(r Runner) Model        { return Model{runner: r, events: make(chan event, 32)} }
func (m Model) Init() tea.Cmd   { return nil }
func (m Model) listen() tea.Cmd { return func() tea.Msg { return <-m.events } }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyPressMsg:
		switch v.String() {
		case "q", "ctrl+c":
			if m.running {
				m.cancel()
				return m, nil
			}
			return m, tea.Quit
		case "enter":
			if m.running {
				return m, nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			m.cancel = cancel
			m.running = true
			m.done = false
			m.err = nil
			runner, events := m.runner, m.events
			return m, tea.Batch(m.listen(), func() tea.Msg {
				err := runner.Run(ctx, func(s string) {
					select {
					case events <- event{text: s}:
					case <-ctx.Done():
					}
				})
				events <- event{done: true, err: err}
				return nil
			})
		}
	case event:
		if v.done {
			m.running = false
			m.done = true
			m.err = v.err
			m.cancel()
			return m, nil
		}
		m.lines = append(m.lines, v.text)
		if len(m.lines) > 15 {
			m.lines = m.lines[len(m.lines)-15:]
		}
		return m, m.listen()
	}
	return m, nil
}
func (m Model) View() tea.View {
	status := "Enter: synchronize   q: exit"
	if m.running {
		status = "Synchronizing… Ctrl+C: cancel"
	} else if m.done {
		status = "Finished. Enter: retry/synchronize   q: exit"
	}
	if m.err != nil {
		status = fmt.Sprintf("%v\n%s", m.err, status)
	}
	return tea.NewView("MusicPort\n\n" + strings.Join(m.lines, "\n") + "\n\n" + status + "\n")
}
