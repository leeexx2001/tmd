package bot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockBot struct {
	started bool
	stopped bool
	name    string
}

func (m *mockBot) Start() error { m.started = true; return nil }
func (m *mockBot) Stop()        { m.stopped = true }
func (m *mockBot) Name() string { return m.name }

func TestBotManager_StartStop(t *testing.T) {
	m1 := &mockBot{name: "mock1"}
	m2 := &mockBot{name: "mock2"}
	bm := NewBotManager(m1, m2)
	bm.Start()
	assert.True(t, m1.started)
	assert.True(t, m2.started)
	bm.Stop()
	assert.True(t, m1.stopped)
	assert.True(t, m2.stopped)
}

func TestBotManager_Empty(t *testing.T) {
	bm := NewBotManager()
	// Should not panic on empty manager
	bm.Start()
	bm.Stop()
}
