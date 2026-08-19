package application

import (
	"context"
	"testing"

	"morsemanual/internal/logbook"
	"morsemanual/internal/settings"
	"morsemanual/internal/stores"
)

func TestTerminalApplicationRegistersLogbookAndDecoderPages(t *testing.T) {
	terminal, err := newTerminalApplication(t.Context(), dependencies{
		stores: stores.Stores{
			Logbook:  emptyLogbookStore{},
			Settings: emptySettingsStore{},
		},
		qrz: unusedQRZService{},
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

type unusedQRZService struct{}

type emptySettingsStore struct {
	settings.Store
}

func (unusedQRZService) ValidateLogin(context.Context, string, string) error {
	return nil
}

func (unusedQRZService) ValidateAPIKey(context.Context, string) error {
	return nil
}

func (emptyLogbookStore) List(
	context.Context,
	logbook.Filter,
) ([]logbook.QSO, error) {
	return nil, nil
}
