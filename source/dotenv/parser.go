package dotenv

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"
)

var expansionRegexp = regexp.MustCompile(`(\\)?(\$)(?:\{([a-zA-Z0-9_]+)(?::-([^}]*))?\}|([a-zA-Z0-9_]+))`)

// Parser parses .env format data into a key-value map.
type Parser struct {
	expandEnv bool
}

// Option configures the parser.
type Option func(*Parser)

// WithExpandEnv enables or disables variable expansion (interpolation). Default is true.
func WithExpandEnv(expand bool) Option {
	return func(p *Parser) {
		p.expandEnv = expand
	}
}

// NewParser creates a new .env parser.
func NewParser(opts ...Option) *Parser {
	p := &Parser{
		expandEnv: true,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Parse reads from an io.Reader and returns key-value pairs.
func (p *Parser) Parse(r io.Reader) (map[string]string, error) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)
	result := make(map[string]string)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Trim leading whitespace
		trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)

		// Skip comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Handle optional "export " prefix
		if strings.HasPrefix(trimmed, "export ") {
			trimmed = strings.TrimPrefix(trimmed, "export ")
			trimmed = strings.TrimLeftFunc(trimmed, unicode.IsSpace)
		}

		// Look for '=' separator
		before, after, ok := strings.Cut(trimmed, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: invalid format, missing '=' in line %q", lineNum, line)
		}

		key := strings.TrimSpace(before)
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key name", lineNum)
		}

		rawValue := after
		val, err := p.parseValue(rawValue, scanner, &lineNum)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		if p.expandEnv {
			val = p.interpolate(val, result)
		}

		result[key] = val
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading dotenv content: %w", err)
	}

	return result, nil
}

// parseValue extracts a single or multi-line value (quoted or unquoted).
func (p *Parser) parseValue(val string, scanner *bufio.Scanner, lineNum *int) (string, error) {
	val = strings.TrimLeftFunc(val, unicode.IsSpace)

	if len(val) == 0 {
		return "", nil
	}

	firstChar := val[0]

	// Double quoted string: "..."
	if firstChar == '"' {
		return p.parseDoubleQuoted(val[1:], scanner, lineNum)
	}

	// Single quoted string: '...'
	if firstChar == '\'' {
		return p.parseSingleQuoted(val[1:], scanner, lineNum)
	}

	// Unquoted value: ends at inline comment '#' or end of line
	// Note: inline comment must be preceded by whitespace (e.g. "PORT=8080 # comment")
	// or unescaped #.
	var sb strings.Builder
	for i := 0; i < len(val); i++ {
		if val[i] == '#' && (i == 0 || unicode.IsSpace(rune(val[i-1]))) {
			break
		}
		sb.WriteByte(val[i])
	}

	return strings.TrimRightFunc(sb.String(), unicode.IsSpace), nil
}

func (p *Parser) parseDoubleQuoted(rest string, scanner *bufio.Scanner, lineNum *int) (string, error) {
	var buf bytes.Buffer
	current := rest

	for {
		escaped := false
		closed := false

		for i := 0; i < len(current); i++ {
			c := current[i]

			if escaped {
				switch c {
				case 'n':
					buf.WriteByte('\n')
				case 'r':
					buf.WriteByte('\r')
				case 't':
					buf.WriteByte('\t')
				case '"':
					buf.WriteByte('"')
				case '\\':
					buf.WriteByte('\\')
				case '$':
					buf.WriteByte('$')
				default:
					buf.WriteByte('\\')
					buf.WriteByte(c)
				}
				escaped = false
				continue
			}

			if c == '\\' {
				escaped = true
				continue
			}

			if c == '"' {
				// String is closed
				closed = true
				// Check remainder of line for trailing comments
				trailing := strings.TrimSpace(current[i+1:])
				if trailing != "" && !strings.HasPrefix(trailing, "#") {
					return "", fmt.Errorf("unexpected characters after closing quote: %q", trailing)
				}
				return buf.String(), nil
			}

			buf.WriteByte(c)
		}

		if closed {
			break
		}

		// If multiline, append newline and scan next line
		buf.WriteByte('\n')
		if !scanner.Scan() {
			return "", fmt.Errorf("unclosed double quote starting at or before line %d", *lineNum)
		}
		*lineNum++
		current = scanner.Text()
	}

	return buf.String(), nil
}

func (p *Parser) parseSingleQuoted(rest string, scanner *bufio.Scanner, lineNum *int) (string, error) {
	var buf bytes.Buffer
	current := rest

	for {
		for i := 0; i < len(current); i++ {
			c := current[i]
			if c == '\'' {
				// String is closed
				trailing := strings.TrimSpace(current[i+1:])
				if trailing != "" && !strings.HasPrefix(trailing, "#") {
					return "", fmt.Errorf("unexpected characters after closing quote: %q", trailing)
				}
				return buf.String(), nil
			}
			buf.WriteByte(c)
		}

		// Multiline single quote
		buf.WriteByte('\n')
		if !scanner.Scan() {
			return "", fmt.Errorf("unclosed single quote starting at or before line %d", *lineNum)
		}
		*lineNum++
		current = scanner.Text()
	}
}

// interpolate expands ${VAR:-default} and $VAR using already parsed values or OS environment.
func (p *Parser) interpolate(input string, currentValues map[string]string) string {
	return expansionRegexp.ReplaceAllStringFunc(input, func(m string) string {
		sub := expansionRegexp.FindStringSubmatch(m)
		if len(sub) == 0 {
			return m
		}

		// If escaped with \, e.g. "\$FOO" or "\${FOO}"
		if sub[1] == "\\" {
			return m[1:] // return without the backslash
		}

		// sub[3] is ${VAR}, sub[4] is default in ${VAR:-default}, sub[5] is $VAR
		varName := sub[3]
		defaultVal := sub[4]
		hasDefault := sub[4] != "" || strings.Contains(m, ":-")

		if varName == "" {
			varName = sub[5]
		}

		if varName == "" {
			return m
		}

		// 1. Check in already parsed dotenv values
		if val, exists := currentValues[varName]; exists && val != "" {
			return val
		}

		// 2. Check in OS environment
		if val, exists := os.LookupEnv(varName); exists && val != "" {
			return val
		}

		// 3. Fallback to default value if provided
		if hasDefault {
			return defaultVal
		}

		return ""
	})
}
