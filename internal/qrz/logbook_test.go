package qrz

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	domain "morsemanual/internal/logbook"
	"morsemanual/internal/optional"
)

func TestUploadQSOUsesReplaceAndEncodesADIF(t *testing.T) {
	client := testClient(t, func(request *http.Request) string {
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if request.Form.Get("KEY") != "api-key" ||
			request.Form.Get("ACTION") != "INSERT" ||
			request.Form.Get("OPTION") != "REPLACE" {
			t.Fatalf("form = %#v", request.Form)
		}
		adif := request.Form.Get("ADIF")
		for _, field := range []string{
			"<BAND:3>40m",
			"<FREQ:6>7.0235",
			"<MODE:2>CW",
			"<CALL:8>DL8ECA/P",
			"<QSO_DATE:8>20260813",
			"<TIME_ON:6>161705",
			"<STATION_CALLSIGN:6>HA7NCS",
			"<COMMENT:5>flood",
			"<EOR>",
		} {
			if !strings.Contains(adif, field) {
				t.Errorf("ADIF %q does not contain %q", adif, field)
			}
		}
		return "RESULT=REPLACE&LOGID=130877825&COUNT=1"
	})
	service := newService(client, "https://xml.example", "https://api.example")
	logID, err := service.UploadQSO(t.Context(), "api-key", domain.QSO{
		StationCallsign: "HA7NCS",
		Callsign:        "DL8ECA/P",
		StartedAt:       time.Date(2026, 8, 13, 16, 17, 5, 0, time.UTC),
		FrequencyHz:     optional.Some[int64](7_023_500),
		Mode:            "CW",
		Notes:           "flood",
	})
	if err != nil {
		t.Fatalf("UploadQSO() error = %v", err)
	}
	if logID != 130877825 {
		t.Fatalf("UploadQSO() log id = %d", logID)
	}
}

func TestUploadQSORequiresFrequency(t *testing.T) {
	service := newService(http.DefaultClient, "https://xml.example", "https://api.example")
	_, err := service.UploadQSO(t.Context(), "api-key", domain.QSO{})
	if err == nil || !strings.Contains(err.Error(), "frequency") {
		t.Fatalf("UploadQSO() error = %v", err)
	}
}

func TestDeleteQSOTreatsMissingRemoteRecordAsComplete(t *testing.T) {
	client := testClient(t, func(request *http.Request) string {
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if request.Form.Get("ACTION") != "DELETE" ||
			request.Form.Get("LOGIDS") != "12345" {
			t.Fatalf("form = %#v", request.Form)
		}
		return "RESULT=FAIL&REASON=logid+not+found&COUNT=0"
	})
	service := newService(client, "https://xml.example", "https://api.example")
	if err := service.DeleteQSO(t.Context(), "api-key", 12345); err != nil {
		t.Fatalf("DeleteQSO() error = %v", err)
	}
}

func TestBandForFrequency(t *testing.T) {
	tests := map[int64]string{
		136_000:        "2190m",
		7_023_500:      "40m",
		144_300_000:    "2m",
		10_368_000_000: "3cm",
		100_000_000:    "",
	}
	for frequency, want := range tests {
		if got := bandForFrequency(frequency); got != want {
			t.Errorf("bandForFrequency(%d) = %q, want %q", frequency, got, want)
		}
	}
}

func TestParseLogbookResponseHasNoQueryParameterLimit(t *testing.T) {
	values := make(url.Values)
	values.Set("RESULT", "OK")
	for index := range 1_100 {
		values.Set(fmt.Sprintf("FIELD_%d", index), "value")
	}
	parsed, err := parseLogbookResponse([]byte(values.Encode()))
	if err != nil {
		t.Fatalf("parseLogbookResponse() error = %v", err)
	}
	if parsed.Get("RESULT") != "OK" || parsed.Get("FIELD_1099") != "value" {
		t.Fatalf("parseLogbookResponse() omitted values")
	}
}
