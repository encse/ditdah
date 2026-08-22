package application

import (
	"context"
	"testing"

	"ditdah/internal/callsign"
	"ditdah/internal/logbook"
	"ditdah/internal/settings"
	"ditdah/internal/stores"
)

func TestTerminalApplicationCreatesInitialLogbookPage(t *testing.T) {
	terminal, initialPage := newTerminalApplication(dependencies{
		stores: stores.Stores{
			Logbook:  emptyLogbookStore{},
			Settings: emptySettingsStore{},
		},
		qrz: unusedQRZService{},
	})
	if terminal == nil {
		t.Fatal("newTerminalApplication() returned nil")
	}
	if initialPage == nil || initialPage.ID() != "logbook" {
		t.Fatalf("initial page = %v, want logbook", initialPage)
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

func (unusedQRZService) LookupCallsign(
	context.Context,
	string,
	string,
	string,
) (callsign.Record, error) {
	return callsign.Record{}, nil
}

func (emptyLogbookStore) List(
	context.Context,
	logbook.Filter,
) ([]logbook.QSO, error) {
	return nil, nil
}
