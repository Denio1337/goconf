package dotenv

import (
	"strings"
	"testing"
)

func TestParserBasic(t *testing.T) {
	input := `
# Comment at top
PORT=8080
HOST=localhost
DEBUG=true
`
	p := NewParser()
	res, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]string{
		"PORT":  "8080",
		"HOST":  "localhost",
		"DEBUG": "true",
	}

	for k, v := range expected {
		if res[k] != v {
			t.Errorf("expected %s=%q, got %q", k, v, res[k])
		}
	}
}

func TestParserExportAndWhitespace(t *testing.T) {
	input := `
export API_KEY = secret123
export   APP_NAME="My Application"
SPACED_VALUE = " value with spaces "
SINGLE_QUOTED = 'single quoted with # hash'
`
	p := NewParser()
	res, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res["API_KEY"] != "secret123" {
		t.Errorf("API_KEY: expected %q, got %q", "secret123", res["API_KEY"])
	}
	if res["APP_NAME"] != "My Application" {
		t.Errorf("APP_NAME: expected %q, got %q", "My Application", res["APP_NAME"])
	}
	if res["SPACED_VALUE"] != " value with spaces " {
		t.Errorf("SPACED_VALUE: expected %q, got %q", " value with spaces ", res["SPACED_VALUE"])
	}
	if res["SINGLE_QUOTED"] != "single quoted with # hash" {
		t.Errorf("SINGLE_QUOTED: expected %q, got %q", "single quoted with # hash", res["SINGLE_QUOTED"])
	}
}

func TestParserEscapesAndMultiline(t *testing.T) {
	input := `
MULTILINE="line 1\nline 2\ttabbed"
CERT="-----BEGIN CERT-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
-----END CERT-----"
SINGLE_MULTILINE='part 1
part 2'
`
	p := NewParser()
	res, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res["MULTILINE"] != "line 1\nline 2\ttabbed" {
		t.Errorf("MULTILINE: expected line break and tab, got %q", res["MULTILINE"])
	}

	expectedCert := "-----BEGIN CERT-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A\n-----END CERT-----"
	if res["CERT"] != expectedCert {
		t.Errorf("CERT: expected %q, got %q", expectedCert, res["CERT"])
	}

	expectedSingle := "part 1\npart 2"
	if res["SINGLE_MULTILINE"] != expectedSingle {
		t.Errorf("SINGLE_MULTILINE: expected %q, got %q", expectedSingle, res["SINGLE_MULTILINE"])
	}
}

func TestParserVariableInterpolation(t *testing.T) {
	input := `
DOMAIN=example.com
PORT=8080
BASE_URL=https://${DOMAIN}:${PORT}/api
BACKUP_URL=https://$DOMAIN:9000
FALLBACK_VAL=${NON_EXISTENT_VAR:-default_val}
ESCAPED_DOLLAR=\$NOT_EXPANDED
`
	p := NewParser(WithExpandEnv(true))
	res, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res["BASE_URL"] != "https://example.com:8080/api" {
		t.Errorf("BASE_URL: expected https://example.com:8080/api, got %q", res["BASE_URL"])
	}
	if res["BACKUP_URL"] != "https://example.com:9000" {
		t.Errorf("BACKUP_URL: expected https://example.com:9000, got %q", res["BACKUP_URL"])
	}
	if res["FALLBACK_VAL"] != "default_val" {
		t.Errorf("FALLBACK_VAL: expected default_val, got %q", res["FALLBACK_VAL"])
	}
	if res["ESCAPED_DOLLAR"] != "$NOT_EXPANDED" {
		t.Errorf("ESCAPED_DOLLAR: expected $NOT_EXPANDED, got %q", res["ESCAPED_DOLLAR"])
	}
}

func TestParserErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing equals", "INVALID_LINE_WITHOUT_EQUALS"},
		{"unclosed quote", `KEY="unclosed string`},
		{"unclosed single quote", `KEY='unclosed single string`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.Parse(strings.NewReader(tc.input))
			if err == nil {
				t.Fatalf("expected error for input %q, got nil", tc.input)
			}
		})
	}
}
