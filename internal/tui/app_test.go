package tui

import (
	"context"
	"testing"

	domain "morsemanual/internal/logbook"
)

func TestAppRegistersOnlyLogbookPage(t *testing.T) {
	app, err := newApp(t.Context(), emptyLogbookStore{})
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	concrete := app.(*application)
	if len(concrete.pages) != 1 {
		t.Fatalf("registered pages = %d, want 1", len(concrete.pages))
	}
	if concrete.activePage == nil || concrete.activePage.ID() != "logbook" {
		t.Fatalf("active page = %v, want logbook", concrete.activePage)
	}
}

type emptyLogbookStore struct {
	domain.Store
}

func (emptyLogbookStore) List(
	context.Context,
	domain.Filter,
) ([]domain.QSO, error) {
	return nil, nil
}
