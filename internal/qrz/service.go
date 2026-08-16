// Package qrz provides access to QRZ.com services.
package qrz

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	xmlEndpoint     = "https://xmldata.qrz.com/xml/current/"
	logbookEndpoint = "https://logbook.qrz.com/api"
	userAgent       = "MorseManual/0.1 (HA7NCS)"
	maxResponseSize = 1 << 20
)

// Service validates credentials used by the QRZ XML and Logbook APIs.
type Service interface {
	ValidateLogin(ctx context.Context, callsign, password string) error
	ValidateAPIKey(ctx context.Context, apiKey string) error
}

type service struct {
	client          *http.Client
	xmlEndpoint     string
	logbookEndpoint string
}

// New creates a QRZ service using the public QRZ endpoints.
func New() Service {
	return newService(
		&http.Client{Timeout: 10 * time.Second},
		xmlEndpoint,
		logbookEndpoint,
	)
}

func newService(client *http.Client, xmlURL, logbookURL string) Service {
	return &service{
		client:          client,
		xmlEndpoint:     xmlURL,
		logbookEndpoint: logbookURL,
	}
}

func (s *service) ValidateLogin(
	ctx context.Context,
	callsign string,
	password string,
) error {
	if strings.TrimSpace(callsign) == "" {
		return errors.New("callsign is required")
	}
	if password == "" {
		return errors.New("password is required")
	}

	body, err := s.post(ctx, s.xmlEndpoint, url.Values{
		"username": {callsign},
		"password": {password},
		"agent":    {userAgent},
	})
	if err != nil {
		return fmt.Errorf("QRZ.com login: %w", err)
	}

	var response xmlResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("QRZ.com login: decode response: %w", err)
	}
	if message := strings.TrimSpace(response.Session.Error); message != "" {
		return fmt.Errorf("QRZ.com login: %s", message)
	}
	if strings.TrimSpace(response.Session.Key) == "" {
		message := strings.TrimSpace(response.Session.Message)
		if message == "" {
			message = "no session key returned"
		}
		return fmt.Errorf("QRZ.com login: %s", message)
	}
	return nil
}

func (s *service) ValidateAPIKey(ctx context.Context, apiKey string) error {
	if strings.TrimSpace(apiKey) == "" {
		return errors.New("API key is required")
	}

	body, err := s.post(ctx, s.logbookEndpoint, url.Values{
		"KEY":    {apiKey},
		"ACTION": {"STATUS"},
	})
	if err != nil {
		return fmt.Errorf("QRZ.com API key: %w", err)
	}
	values, parseErr := url.ParseQuery(string(body))
	if parseErr != nil {
		return fmt.Errorf("QRZ.com API key: decode response: %w", parseErr)
	}
	if strings.EqualFold(values.Get("RESULT"), "OK") {
		return nil
	}
	reason := strings.TrimSpace(values.Get("REASON"))
	if reason == "" {
		result := strings.TrimSpace(values.Get("RESULT"))
		if result == "" {
			reason = "missing result"
		} else {
			reason = "server returned " + result
		}
	}
	return fmt.Errorf("QRZ.com API key: %s", reason)
}

func (s *service) post(
	ctx context.Context,
	endpoint string,
	values url.Values,
) ([]byte, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", userAgent)

	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("server returned %s", response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize))
	if err != nil {
		return nil, err
	}
	return body, nil
}

type xmlResponse struct {
	Session struct {
		Key     string `xml:"Key"`
		Message string `xml:"Message"`
		Error   string `xml:"Error"`
	} `xml:"Session"`
}
