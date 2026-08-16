package application

import (
	"context"
	"testing"

	"morsemanual/internal/logbook"
	"morsemanual/internal/stores"
)

func TestTerminalApplicationRegistersOnlyLogbookPage(t *testing.T) {
	terminal, err := newTerminalApplication(t.Context(), stores.Stores{
		Logbook: emptyLogbookStore{},
	})
	if err != nil {
		t.Fatalf("newTerminalApplication() error = %v", err)
	}
	if terminal == nil {
		t.Fatal("newTerminalApplication() returned nil")
	}
}

type emptyLogbookStore struct {
	logbook.Store
}

func (emptyLogbookStore) List(
	context.Context,
	logbook.Filter,
) ([]logbook.QSO, error) {
	return nil, nil
}
