package qrz

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestValidateLogin(t *testing.T) {
	client := testClient(t, func(request *http.Request) string {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if request.UserAgent() != userAgent {
			t.Fatalf("user agent = %q", request.UserAgent())
		}
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if request.Form.Get("username") != "HA7NCS" ||
			request.Form.Get("password") != "secret" {
			t.Fatalf("form = %#v", request.Form)
		}
		return `<?xml version="1.0"?>
<QRZDatabase><Session><Key>session-key</Key></Session></QRZDatabase>`
	})

	service := newService(client, "https://xml.example", "https://api.example")
	if err := service.ValidateLogin(t.Context(), "HA7NCS", "secret"); err != nil {
		t.Fatalf("ValidateLogin() error = %v", err)
	}
}

func TestValidateLoginReturnsQRZError(t *testing.T) {
	client := testClient(t, func(*http.Request) string {
		return `<QRZDatabase><Session>` +
			`<Error>password incorrect</Error></Session></QRZDatabase>`
	})

	service := newService(client, "https://xml.example", "https://api.example")
	err := service.ValidateLogin(t.Context(), "HA7NCS", "wrong")
	if err == nil || !strings.Contains(err.Error(), "password incorrect") {
		t.Fatalf("ValidateLogin() error = %v", err)
	}
}

func TestValidateAPIKey(t *testing.T) {
	client := testClient(t, func(request *http.Request) string {
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if request.Form.Get("KEY") != "valid-key" ||
			request.Form.Get("ACTION") != "STATUS" {
			t.Fatalf("form = %#v", request.Form)
		}
		return "RESULT=OK&DATA=BOOKID%3D42"
	})

	service := newService(client, "https://xml.example", "https://api.example")
	if err := service.ValidateAPIKey(t.Context(), "valid-key"); err != nil {
		t.Fatalf("ValidateAPIKey() error = %v", err)
	}
}

func TestValidateAPIKeyReturnsReason(t *testing.T) {
	client := testClient(t, func(*http.Request) string {
		return "RESULT=FAIL&REASON=invalid+key"
	})

	service := newService(client, "https://xml.example", "https://api.example")
	err := service.ValidateAPIKey(context.Background(), "wrong-key")
	if err == nil || !strings.Contains(err.Error(), "invalid key") {
		t.Fatalf("ValidateAPIKey() error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func testClient(
	t *testing.T,
	responseBody func(*http.Request) string,
) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(
		request *http.Request,
	) (*http.Response, error) {
		body := responseBody(request)
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})}
}
