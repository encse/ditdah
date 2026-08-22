package qrz

import (
	"errors"
	"net/http"
	"testing"

	"ditdah/internal/callsign"
)

func TestLookupCallsignLogsInAndMapsQRZRecord(t *testing.T) {
	requests := 0
	client := testClient(t, func(request *http.Request) string {
		requests++
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if requests == 1 {
			if request.Form.Get("username") != "HA7NCS" ||
				request.Form.Get("password") != "secret" {
				t.Fatalf("login form = %#v", request.Form)
			}
			return `<QRZDatabase><Session><Key>session-key</Key></Session></QRZDatabase>`
		}
		if request.Form.Get("s") != "session-key" ||
			request.Form.Get("callsign") != "DL1DAW" {
			t.Fatalf("lookup form = %#v", request.Form)
		}
		return `<QRZDatabase><Callsign>` +
			`<call>DL1DAW</call><fname>Jane</fname><name>Doe</name>` +
			`<nickname>JD</nickname><addr2>Berlin</addr2>` +
			`<state>BE</state><country>Germany</country><grid>JO62</grid>` +
			`<cqzone>14</cqzone><ituzone>28</ituzone>` +
			`<url>https://www.qrz.com/db/DL1DAW</url>` +
			`</Callsign></QRZDatabase>`
	})
	service := newService(client, "https://xml.example", "https://api.example")

	record, err := service.LookupCallsign(
		t.Context(), "HA7NCS", "secret", "dl1daw",
	)
	if err != nil {
		t.Fatalf("LookupCallsign() error = %v", err)
	}
	if record.Callsign != "DL1DAW" || optionalValue(record.Name) != "Jane Doe" ||
		optionalValue(record.Country) != "Germany" ||
		optionalValue(record.Grid) != "JO62" ||
		optionalValue(record.QTH) != "Berlin" {
		t.Fatalf("LookupCallsign() = %#v", record)
	}
}

func TestLookupCallsignMapsNotFoundToProviderSentinel(t *testing.T) {
	requests := 0
	client := testClient(t, func(*http.Request) string {
		requests++
		if requests == 1 {
			return `<QRZDatabase><Session><Key>session-key</Key></Session></QRZDatabase>`
		}
		return `<QRZDatabase><Session><Error>Not found: N0PE</Error></Session></QRZDatabase>`
	})
	service := newService(client, "https://xml.example", "https://api.example")

	_, err := service.LookupCallsign(t.Context(), "HA7NCS", "secret", "N0PE")
	if !errors.Is(err, callsign.ErrProviderNotFound) {
		t.Fatalf("LookupCallsign() error = %v, want ErrProviderNotFound", err)
	}
}

func optionalValue(value interface{ Get() (string, bool) }) string {
	result, _ := value.Get()
	return result
}
