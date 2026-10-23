package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"github.com/gojuno/minimock/v3"
	"testing"
)

func TestStartAndCancel(t *testing.T) {
	runner := NewRunnerMock(minimock.NewController(t))
	model := New(runner)
	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m := next.(Model)
	if cmd == nil || !m.running {
		t.Fatal("not started")
	}
	_, second := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if second != nil {
		t.Fatal("second run started")
	}
	m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	m.cancel()
}
func TestCompletion(t *testing.T) {
	m := New(NewRunnerMock(minimock.NewController(t)))
	_, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.running = true
	next, _ := m.Update(event{done: true})
	got := next.(Model)
	if got.running || !got.done {
		t.Fatal("not completed")
	}
}
