package qrz

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"

	domain "ditdah/internal/callsign"
	"ditdah/internal/optional"
)

type xmlCallsign struct {
	Call     string `xml:"call"`
	First    string `xml:"fname"`
	Last     string `xml:"name"`
	Nickname string `xml:"nickname"`
	QTH      string `xml:"addr2"`
	State    string `xml:"state"`
	Country  string `xml:"country"`
	Grid     string `xml:"grid"`
	CQZone   string `xml:"cqzone"`
	ITUZone  string `xml:"ituzone"`
	URL      string `xml:"url"`
}

func (s *service) LookupCallsign(
	ctx context.Context,
	username string,
	password string,
	callsign string,
) (domain.Record, error) {
	callsign = strings.ToUpper(strings.TrimSpace(callsign))
	if callsign == "" {
		return domain.Record{}, errors.New("callsign is required")
	}
	sessionKey, err := s.login(ctx, username, password)
	if err != nil {
		return domain.Record{}, err
	}

	body, err := s.post(ctx, s.xmlEndpoint, url.Values{
		"s":        {sessionKey},
		"callsign": {callsign},
	})
	if err != nil {
		return domain.Record{}, fmt.Errorf("QRZ.com callsign lookup: %w", err)
	}
	var response xmlResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		return domain.Record{}, fmt.Errorf(
			"QRZ.com callsign lookup: decode response: %w",
			err,
		)
	}
	if message := strings.TrimSpace(response.Session.Error); message != "" {
		if strings.Contains(strings.ToLower(message), "not found") {
			return domain.Record{}, fmt.Errorf(
				"%w: %s",
				domain.ErrProviderNotFound,
				message,
			)
		}
		return domain.Record{}, fmt.Errorf("QRZ.com callsign lookup: %s", message)
	}
	if strings.TrimSpace(response.Callsign.Call) == "" {
		return domain.Record{}, errors.New(
			"QRZ.com callsign lookup: no callsign record returned",
		)
	}
	return recordFromXML(response.Callsign), nil
}

func recordFromXML(value xmlCallsign) domain.Record {
	return domain.Record{
		Callsign: strings.ToUpper(strings.TrimSpace(value.Call)),
		Country:  optionalText(value.Country),
		CQZone:   optionalText(value.CQZone),
		Grid:     optionalText(value.Grid),
		ITUZone:  optionalText(value.ITUZone),
		Name:     optionalText(joinName(value.First, value.Last)),
		Nickname: optionalText(value.Nickname),
		QRZURL:   optionalText(value.URL),
		QTH:      optionalText(value.QTH),
		State:    optionalText(value.State),
	}
}

func optionalText(value string) optional.Value[string] {
	value = strings.TrimSpace(value)
	if value == "" {
		return optional.None[string]()
	}
	return optional.Some(value)
}

func joinName(first, last string) string {
	return strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
}
