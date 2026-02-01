// Package gui provides undo/redo functionality for CPC screen editing.
package gui

import "github.com/ikari-pl/go-cpc-image/pkg/render"

const maxUndoLevels = 50

// Snapshot captures the CPC screen state for undo/redo.
type Snapshot struct {
	ScreenData [0x10000]byte
	Palette    [16]int
}

// UndoStack manages undo/redo history as a ring of snapshots.
type UndoStack struct {
	states []Snapshot
	pos    int // index of next free slot (also current position)
	count  int // number of undo-able states behind pos
	redos  int // number of redo-able states ahead of pos
}

// NewUndoStack creates a new empty undo stack.
func NewUndoStack() *UndoStack {
	return &UndoStack{
		states: make([]Snapshot, 0, maxUndoLevels),
	}
}

// Push saves a new snapshot, discarding any redo history.
func (u *UndoStack) Push(s Snapshot) {
	if len(u.states) < maxUndoLevels {
		// Still growing the slice
		u.states = append(u.states, Snapshot{})
	}
	u.states[u.pos%maxUndoLevels] = s
	u.pos = (u.pos + 1) % maxUndoLevels
	if u.count < maxUndoLevels-1 {
		u.count++
	}
	u.redos = 0
}

// CanUndo returns true if there is at least one state to undo to.
func (u *UndoStack) CanUndo() bool {
	return u.count > 0
}

// CanRedo returns true if there is at least one state to redo to.
func (u *UndoStack) CanRedo() bool {
	return u.redos > 0
}

// Undo returns the previous snapshot, or nil if nothing to undo.
func (u *UndoStack) Undo() *Snapshot {
	if !u.CanUndo() {
		return nil
	}
	u.pos = (u.pos - 1 + maxUndoLevels) % maxUndoLevels
	u.count--
	u.redos++
	s := u.states[u.pos]
	return &s
}

// Redo returns the next snapshot, or nil if nothing to redo.
func (u *UndoStack) Redo() *Snapshot {
	if !u.CanRedo() {
		return nil
	}
	s := u.states[u.pos]
	u.pos = (u.pos + 1) % maxUndoLevels
	u.count++
	u.redos--
	return &s
}

// CaptureSnapshot creates a snapshot from the current BitmapCpc state.
func CaptureSnapshot(b *render.BitmapCpc) Snapshot {
	var s Snapshot
	s.ScreenData = b.ScreenData
	copy(s.Palette[:], b.Palette[:])
	return s
}

// RestoreSnapshot applies a snapshot to a BitmapCpc.
func RestoreSnapshot(b *render.BitmapCpc, s *Snapshot) {
	b.ScreenData = s.ScreenData
	copy(b.Palette[:], s.Palette[:])
}
