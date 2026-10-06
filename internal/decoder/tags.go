package decoder

import (
	"reflect"
	"strings"
	"unicode"
)

// Canonical struct tag constants used across the library for schema definition.
const (
	TagKey         = "key"
	TagDefault     = "default"
	TagRequired    = "required"
	TagPrefix      = "prefix"
	TagSep         = "sep"
	TagLayout      = "layout"
	TagSecret      = "secret"
)

type fieldTagInfo struct {
	primaryKey   string
	required     bool
	hasDefault   bool
	defaultValue string
	separator    string
	layout       string
	isSecret     bool
}

func parseFieldTag(field reflect.StructField) fieldTagInfo {
	info := fieldTagInfo{
		separator: ",",
	}

	// 1. Check TagKey ("key")
	tag := field.Tag.Get(TagKey)
	if tag != "" {
		parts := strings.Split(tag, ",")
		if len(parts) > 0 && parts[0] != "" {
			info.primaryKey = parts[0]
		}
		for _, opt := range parts[1:] {
			opt = strings.TrimSpace(opt)
			if opt == "required" {
				info.required = true
			} else if strings.HasPrefix(opt, "default=") {
				info.hasDefault = true
				info.defaultValue = strings.TrimPrefix(opt, "default=")
			}
		}
	}

	// 2. Check explicit default tag
	if def, ok := field.Tag.Lookup(TagDefault); ok {
		info.hasDefault = true
		info.defaultValue = def
	}

	// 3. Check explicit required tag
	if req, ok := field.Tag.Lookup(TagRequired); ok {
		info.required = strings.EqualFold(req, "true") || req == "1"
	}

	// 4. Separator tag for slices
	if sep := field.Tag.Get(TagSep); sep != "" {
		info.separator = sep
	}

	// 5. Layout tag for time.Time
	if layout := field.Tag.Get(TagLayout); layout != "" {
		info.layout = layout
	}

	// 6. Secret tag or SecretMarker wrapper
	if sec, ok := field.Tag.Lookup(TagSecret); ok {
		info.isSecret = strings.EqualFold(sec, "true") || sec == "1"
	}
	if isSecretType(field.Type) {
		info.isSecret = true
	}

	return info
}

func (d *Decoder) buildCandidateKeys(tagInfo fieldTagInfo, prefix, fieldName string) []string {
	var candidates []string

	if tagInfo.primaryKey != "" {
		// Absolute key (starts with '/'): ignore all parent prefixes!
		if strings.HasPrefix(tagInfo.primaryKey, "/") {
			absKey := strings.TrimPrefix(tagInfo.primaryKey, "/")
			return []string{absKey}
		}

		if prefix != "" {
			candidates = append(candidates, prefix+tagInfo.primaryKey)
			if !strings.HasSuffix(prefix, "_") && !strings.HasSuffix(prefix, ".") {
				candidates = append(candidates, prefix+"_"+tagInfo.primaryKey)
				candidates = append(candidates, prefix+"."+tagInfo.primaryKey)
			}
		} else {
			candidates = append(candidates, tagInfo.primaryKey)
		}
	}

	// Default naming conventions derived from field name
	snake := toScreamingSnake(fieldName)
	if prefix != "" {
		candidates = append(candidates, prefix+snake)
		if !strings.HasSuffix(prefix, "_") && !strings.HasSuffix(prefix, ".") {
			candidates = append(candidates, prefix+"_"+snake)
		}
		candidates = append(candidates, prefix+fieldName)
	} else {
		candidates = append(candidates, snake)
		candidates = append(candidates, fieldName)
	}

	// Additional lower/dot forms
	lowerSnake := strings.ToLower(snake)
	if prefix != "" {
		cleanPrefix := strings.TrimRight(prefix, "_.")
		lowerPrefix := strings.ToLower(cleanPrefix)
		candidates = append(candidates, lowerPrefix+"."+lowerSnake)
		candidates = append(candidates, lowerPrefix+"_"+lowerSnake)
		candidates = append(candidates, cleanPrefix+"__"+snake)
	} else {
		candidates = append(candidates, lowerSnake)
	}

	return candidates
}

// toScreamingSnake converts camelCase / PascalCase to SCREAMING_SNAKE_CASE.
func toScreamingSnake(s string) string {
	var sb strings.Builder
	sb.Grow(len(s) + 5)
	runes := []rune(s)
	n := len(runes)

	for i := range n {
		r := runes[i]
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			if unicode.IsLower(prev) || unicode.IsDigit(prev) ||
				(i+1 < n && unicode.IsLower(runes[i+1])) {
				sb.WriteByte('_')
			}
		}
		sb.WriteRune(unicode.ToUpper(r))
	}

	return sb.String()
}
