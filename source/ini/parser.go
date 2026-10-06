package ini

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// Parser parses INI formatted configuration data into a hierarchical map.
type Parser struct{}

// NewParser creates a new INI parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads from an io.Reader and returns parsed configuration data.
// Sections are represented as nested map[string]any.
func (p *Parser) Parse(r io.Reader) (map[string]any, error) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)
	result := make(map[string]any)
	currentSection := ""

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and full line comments (; and #)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// Section header: [section] or [section.subsection]
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, fmt.Errorf("line %d: unclosed section header: %q", lineNum, line)
			}
			secName := strings.TrimSpace(line[1 : len(line)-1])
			if secName == "" {
				return nil, fmt.Errorf("line %d: empty section name", lineNum)
			}
			currentSection = secName
			continue
		}

		// Key-value pair separated by '=' or ':'
		idx := strings.IndexAny(line, "=:")
		if idx < 0 {
			return nil, fmt.Errorf("line %d: missing '=' or ':' in line %q", lineNum, line)
		}

		key := strings.TrimSpace(line[:idx])
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key name", lineNum)
		}

		valStr := strings.TrimSpace(line[idx+1:])
		val, err := parseIniValue(valStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		if currentSection == "" {
			result[key] = val
		} else {
			setSectionKey(result, currentSection, key, val)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading INI: %w", err)
	}

	return result, nil
}

func parseIniValue(val string) (string, error) {
	if len(val) == 0 {
		return "", nil
	}

	first := val[0]

	// Double quoted string
	if first == '"' {
		return parseQuoted(val, '"')
	}

	// Single quoted string
	if first == '\'' {
		return parseQuoted(val, '\'')
	}

	// Unquoted: strip trailing inline comments starting with ';' or '#'
	var sb strings.Builder
	for i := 0; i < len(val); i++ {
		c := val[i]
		if (c == ';' || c == '#') && (i == 0 || unicode.IsSpace(rune(val[i-1]))) {
			break
		}
		sb.WriteByte(c)
	}

	return strings.TrimSpace(sb.String()), nil
}

func parseQuoted(val string, quote byte) (string, error) {
	var buf bytes.Buffer
	escaped := false
	closed := false

	for i := 1; i < len(val); i++ {
		c := val[i]

		if escaped {
			switch c {
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			case '\\':
				buf.WriteByte('\\')
			case quote:
				buf.WriteByte(quote)
			default:
				buf.WriteByte('\\')
				buf.WriteByte(c)
			}
			escaped = false
			continue
		}

		if c == '\\' && quote == '"' {
			escaped = true
			continue
		}

		if c == quote {
			closed = true
			// Trailing comment check
			trailing := strings.TrimSpace(val[i+1:])
			if trailing != "" && !strings.HasPrefix(trailing, ";") && !strings.HasPrefix(trailing, "#") {
				return "", fmt.Errorf("unexpected characters after closing quote: %q", trailing)
			}
			return buf.String(), nil
		}

		buf.WriteByte(c)
	}

	if !closed {
		return "", fmt.Errorf("unclosed quote: %q", val)
	}

	return buf.String(), nil
}

func setSectionKey(root map[string]any, section, key, val string) {
	parts := strings.Split(section, ".")
	current := root

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if sub, ok := current[part]; ok {
			if subMap, isMap := sub.(map[string]any); isMap {
				current = subMap
			} else {
				newMap := make(map[string]any)
				current[part] = newMap
				current = newMap
			}
		} else {
			newMap := make(map[string]any)
			current[part] = newMap
			current = newMap
		}
	}

	current[key] = val
}
