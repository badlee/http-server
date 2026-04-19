package binder

import (
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPDFMiddleware_Basic(t *testing.T) {
	cfg := parseTestBind(t, `
HTTP :8080 BEGIN
	GET /pdf @PDF BEGIN
		return "<h1>Hello PDF</h1>"
	END GET
END HTTP
`)
	dir := getHTTPDir(t, cfg)
	app := dir.App.App

	req := httptest.NewRequest("GET", "/pdf", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	if resp.Header.Get("Content-Type") != "application/pdf" {
		t.Errorf("expected Content-Type application/pdf, got %s", resp.Header.Get("Content-Type"))
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.HasPrefix(string(body), "%PDF-") {
		t.Errorf("expected PDF prefix %%PDF-, got %q", string(body)[:10])
	}
}

func TestPDFMiddleware_Arguments(t *testing.T) {
	cfg := parseTestBind(t, `
HTTP :8080 BEGIN
	GET /pdf-args @PDF[title="My Report" author="Antigravity" name="report" creator="Antigravity" producer="Antigravity" subject="Antigravity" keywords="Antigravity"] BEGIN
		return "Content Antigravity <b>generated content"
	END GET

	GET /pdf-landscape @PDF[orientation=L format=A5] BEGIN
		return "Landscape"
	END GET
END HTTP
`)
	dir := getHTTPDir(t, cfg)
	app := dir.App.App

	t.Run("Metadata and Name", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pdf-args", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}

		if resp.Header.Get("Content-Type") != "application/pdf" {
			t.Errorf("expected application/pdf, got %s", resp.Header.Get("Content-Type"))
		}

		disp := resp.Header.Get("Content-Disposition")
		if !strings.Contains(disp, `filename="report.pdf"`) {
			t.Errorf("expected filename report.pdf in Content-Disposition, got %s", disp)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.HasPrefix(string(body), "%PDF-") {
			t.Errorf("invalid PDF body")
		}
		os.WriteFile("/Users/hobb/.gemini/antigravity/brain/b3228cc5-e64c-4c28-acf3-33bb91f7f562/test.pdf", body, 0644)
		t.Fail()
	})

	t.Run("Orientation and Format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pdf-landscape", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.HasPrefix(string(body), "%PDF-") {
			t.Errorf("invalid PDF body")
		}
	})
}
