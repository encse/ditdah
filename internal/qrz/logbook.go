package qrz

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	domain "ditdah/internal/logbook"
)

// Logbook uploads and removes QRZ Logbook records.
type Logbook interface {
	UploadQSO(ctx context.Context, apiKey string, qso domain.QSO) (int64, error)
	DeleteQSO(ctx context.Context, apiKey string, logID int64) error
}

// Client provides the QRZ XML and Logbook API operations.
type Client interface {
	Service
	Logbook
}

type bandRange struct {
	name    string
	lowerHz int64
	upperHz int64
}

var adifBands = []bandRange{
	{name: "2190m", lowerHz: 135_700, upperHz: 137_800},
	{name: "630m", lowerHz: 472_000, upperHz: 479_000},
	{name: "560m", lowerHz: 501_000, upperHz: 504_000},
	{name: "160m", lowerHz: 1_800_000, upperHz: 2_000_000},
	{name: "80m", lowerHz: 3_500_000, upperHz: 4_000_000},
	{name: "60m", lowerHz: 5_060_000, upperHz: 5_450_000},
	{name: "40m", lowerHz: 7_000_000, upperHz: 7_300_000},
	{name: "30m", lowerHz: 10_100_000, upperHz: 10_150_000},
	{name: "20m", lowerHz: 14_000_000, upperHz: 14_350_000},
	{name: "17m", lowerHz: 18_068_000, upperHz: 18_168_000},
	{name: "15m", lowerHz: 21_000_000, upperHz: 21_450_000},
	{name: "12m", lowerHz: 24_890_000, upperHz: 24_990_000},
	{name: "10m", lowerHz: 28_000_000, upperHz: 29_700_000},
	{name: "8m", lowerHz: 40_000_000, upperHz: 45_000_000},
	{name: "6m", lowerHz: 50_000_000, upperHz: 54_000_000},
	{name: "5m", lowerHz: 54_000_001, upperHz: 69_900_000},
	{name: "4m", lowerHz: 70_000_000, upperHz: 71_000_000},
	{name: "2m", lowerHz: 144_000_000, upperHz: 148_000_000},
	{name: "1.25m", lowerHz: 222_000_000, upperHz: 225_000_000},
	{name: "70cm", lowerHz: 420_000_000, upperHz: 450_000_000},
	{name: "33cm", lowerHz: 902_000_000, upperHz: 928_000_000},
	{name: "23cm", lowerHz: 1_240_000_000, upperHz: 1_300_000_000},
	{name: "13cm", lowerHz: 2_300_000_000, upperHz: 2_450_000_000},
	{name: "9cm", lowerHz: 3_300_000_000, upperHz: 3_500_000_000},
	{name: "6cm", lowerHz: 5_650_000_000, upperHz: 5_925_000_000},
	{name: "3cm", lowerHz: 10_000_000_000, upperHz: 10_500_000_000},
	{name: "1.25cm", lowerHz: 24_000_000_000, upperHz: 24_250_000_000},
	{name: "6mm", lowerHz: 47_000_000_000, upperHz: 47_200_000_000},
	{name: "4mm", lowerHz: 75_500_000_000, upperHz: 81_000_000_000},
	{name: "2.5mm", lowerHz: 119_980_000_000, upperHz: 123_000_000_000},
	{name: "2mm", lowerHz: 134_000_000_000, upperHz: 149_000_000_000},
	{name: "1mm", lowerHz: 241_000_000_000, upperHz: 250_000_000_000},
	{name: "submm", lowerHz: 300_000_000_000, upperHz: 7_500_000_000_000},
}

func (s *service) UploadQSO(
	ctx context.Context,
	apiKey string,
	qso domain.QSO,
) (int64, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return 0, errors.New("API key is required")
	}
	adif, err := encodeQSO(qso)
	if err != nil {
		return 0, err
	}
	body, err := s.post(ctx, s.logbookEndpoint, url.Values{
		"KEY":    {apiKey},
		"ACTION": {"INSERT"},
		"OPTION": {"REPLACE"},
		"ADIF":   {adif},
	})
	if err != nil {
		return 0, fmt.Errorf("QRZ.com upload: %w", err)
	}
	values, err := parseLogbookResponse(body)
	if err != nil {
		return 0, fmt.Errorf("QRZ.com upload: %w", err)
	}
	result := strings.ToUpper(values.Get("RESULT"))
	if result != "OK" && result != "REPLACE" {
		return 0, logbookResponseError("upload", values)
	}
	logIDText := values.Get("LOGID")
	if logIDText == "" {
		logIDText = values.Get("LOGIDS")
	}
	logID, err := strconv.ParseInt(strings.TrimSpace(logIDText), 10, 64)
	if err != nil || logID <= 0 {
		return 0, fmt.Errorf("QRZ.com upload: invalid log id %q", logIDText)
	}
	return logID, nil
}

func (s *service) DeleteQSO(
	ctx context.Context,
	apiKey string,
	logID int64,
) error {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return errors.New("API key is required")
	}
	if logID <= 0 {
		return errors.New("QRZ log id must be positive")
	}
	body, err := s.post(ctx, s.logbookEndpoint, url.Values{
		"KEY":    {apiKey},
		"ACTION": {"DELETE"},
		"LOGIDS": {strconv.FormatInt(logID, 10)},
	})
	if err != nil {
		return fmt.Errorf("QRZ.com delete: %w", err)
	}
	values, err := parseLogbookResponse(body)
	if err != nil {
		return fmt.Errorf("QRZ.com delete: %w", err)
	}
	switch strings.ToUpper(values.Get("RESULT")) {
	case "OK", "PARTIAL":
		return nil
	case "FAIL":
		// For DELETE, QRZ documents FAIL as meaning that no requested LOGID
		// exists. Treat that as success so interrupted replacements can retry.
		return nil
	default:
		return logbookResponseError("delete", values)
	}
}

func encodeQSO(qso domain.QSO) (string, error) {
	frequency, present := qso.FrequencyHz.Get()
	if !present {
		return "", errors.New("QSO frequency is required for QRZ.com")
	}
	band := bandForFrequency(frequency)
	if band == "" {
		return "", fmt.Errorf("frequency %d Hz is outside the ADIF bands", frequency)
	}
	if strings.TrimSpace(qso.StationCallsign) == "" ||
		strings.TrimSpace(qso.Callsign) == "" || qso.StartedAt.IsZero() ||
		strings.TrimSpace(qso.Mode) == "" {
		return "", errors.New("QSO is missing a QRZ.com required field")
	}

	startedAt := qso.StartedAt.UTC()
	fields := [][2]string{
		{"BAND", band},
		{"FREQ", formatMHz(frequency)},
		{"MODE", qso.Mode},
		{"SUBMODE", qso.Submode},
		{"CALL", qso.Callsign},
		{"QSO_DATE", startedAt.Format("20060102")},
		{"TIME_ON", startedAt.Format("150405")},
		{"STATION_CALLSIGN", qso.StationCallsign},
		{"RST_SENT", qso.RSTSent},
		{"RST_RCVD", qso.RSTReceived},
		{"STX_STRING", qso.ExchangeSent},
		{"SRX_STRING", qso.ExchangeReceived},
		{"NAME", qso.Name},
		{"QTH", qso.QTH},
		{"COMMENT", qso.Notes},
	}
	var result strings.Builder
	for _, field := range fields {
		if field[1] == "" {
			continue
		}
		fmt.Fprintf(&result, "<%s:%d>%s", field[0], len(field[1]), field[1])
	}
	result.WriteString("<EOR>")
	return result.String(), nil
}

func bandForFrequency(frequencyHz int64) string {
	for _, band := range adifBands {
		if frequencyHz >= band.lowerHz && frequencyHz <= band.upperHz {
			return band.name
		}
	}
	return ""
}

func formatMHz(frequencyHz int64) string {
	whole := frequencyHz / 1_000_000
	fraction := frequencyHz % 1_000_000
	formatted := fmt.Sprintf("%d.%06d", whole, fraction)
	return strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
}

func parseLogbookResponse(body []byte) (url.Values, error) {
	values := make(url.Values)
	for remaining := string(body); remaining != ""; {
		parameter, rest, found := strings.Cut(remaining, "&")
		remaining = rest
		key, value, _ := strings.Cut(parameter, "=")
		decodedKey, err := url.QueryUnescape(key)
		if err != nil {
			return nil, fmt.Errorf("decode response key: %w", err)
		}
		decodedValue, err := url.QueryUnescape(value)
		if err != nil {
			return nil, fmt.Errorf("decode response value for %q: %w", decodedKey, err)
		}
		values[decodedKey] = append(values[decodedKey], decodedValue)
		if !found {
			break
		}
	}
	if strings.TrimSpace(values.Get("RESULT")) == "" {
		return nil, errors.New("decode response: missing result")
	}
	return values, nil
}

func logbookResponseError(action string, values url.Values) error {
	reason := strings.TrimSpace(values.Get("REASON"))
	if reason == "" {
		reason = "server returned " + strings.TrimSpace(values.Get("RESULT"))
	}
	return fmt.Errorf("QRZ.com %s: %s", action, reason)
}
