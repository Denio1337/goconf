package goenv

import (
	"encoding"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Denio1337/goenv/store"
)

// Validator is an optional interface that structs or fields can implement
// to execute custom business-level validation logic after decoding.
type Validator interface {
	Validate() error
}

// Decoder decodes configuration from a Store into a target struct with strict schema validation.
type Decoder struct {
	store         *store.Store
	strictUnknown bool
	consumedKeys  map[string]bool
}

// NewDecoder creates a new Decoder configured with the provided Store.
func NewDecoder(store *store.Store) *Decoder {
	return &Decoder{
		store:        store,
		consumedKeys: make(map[string]bool),
	}
}

// SetStrictUnknown enables or disables error reporting for keys present in the store but not in the struct.
func (d *Decoder) SetStrictUnknown(strict bool) {
	d.strictUnknown = strict
}

// Decode populates target with data from the Store.
// target must be a non-nil pointer to a struct.
func (d *Decoder) Decode(target any) error {
	if target == nil {
		return ErrInvalidTarget
	}

	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return ErrInvalidTarget
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return ErrInvalidTarget
	}

	valErr := &ValidationError{}
	d.decodeStruct(elem, "", "", valErr)

	// Check if the root struct itself implements Validator
	d.checkValidator(val, "", valErr)

	// If strict unknown keys enabled, check for unused keys
	if d.strictUnknown {
		for key := range d.store.All() {
			if !d.consumedKeys[strings.ToUpper(key)] && !d.consumedKeys[strings.ToLower(key)] && !d.consumedKeys[key] {
				valErr.Add(FieldError{
					Key: key,
					Err: fmt.Errorf("unknown configuration key %q", key),
				})
			}
		}
	}

	if valErr.HasErrors() {
		return valErr
	}
	return nil
}

func (d *Decoder) decodeStruct(v reflect.Value, prefix string, structPath string, valErr *ValidationError) {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		// Ignore unexported fields
		if !field.IsExported() {
			continue
		}

		fieldPath := field.Name
		if structPath != "" {
			fieldPath = structPath + "." + field.Name
		}

		// Handle embedded or nested configuration struct
		if isConfigStruct(field.Type) {
			childPrefix := prefix
			if structPrefix := field.Tag.Get("env-prefix"); structPrefix != "" {
				childPrefix = prefix + structPrefix
			} else if structPrefix := field.Tag.Get("prefix"); structPrefix != "" {
				childPrefix = prefix + structPrefix
			} else if !field.Anonymous {
				childPrefix = prefix + toScreamingSnake(field.Name) + "_"
			}

			if field.Type.Kind() == reflect.Pointer {
				if fieldVal.IsNil() {
					fieldVal.Set(reflect.New(field.Type.Elem()))
				}
				d.decodeStruct(fieldVal.Elem(), childPrefix, fieldPath, valErr)
				d.checkValidator(fieldVal, fieldPath, valErr)
			} else {
				d.decodeStruct(fieldVal, childPrefix, fieldPath, valErr)
				d.checkValidator(fieldVal.Addr(), fieldPath, valErr)
			}
			continue
		}

		// Parse tags and options
		tagInfo := parseFieldTag(field, prefix)

		// Candidate keys for lookup in the store
		candidates := d.buildCandidateKeys(tagInfo, prefix, field.Name)

		// Look up value in store
		rawVal, matchedKey, found := d.store.Get(candidates...)
		if found {
			d.consumedKeys[matchedKey] = true
			d.consumedKeys[strings.ToUpper(matchedKey)] = true
		}

		// Check for missing value
		var valStr string
		hasValue := false

		if found && rawVal != nil {
			valStr = fmt.Sprint(rawVal)
			hasValue = strings.TrimSpace(valStr) != ""
		}

		// Fallback to default if empty or not found
		if !hasValue && tagInfo.hasDefault {
			valStr = tagInfo.defaultValue
			hasValue = true
		}

		// If still missing, check required constraint
		if !hasValue {
			if tagInfo.required {
				primaryKey := ""
				if len(candidates) > 0 {
					primaryKey = candidates[0]
				} else if tagInfo.primaryKey != "" {
					primaryKey = tagInfo.primaryKey
				}
				valErr.Add(FieldError{
					Field:      fieldPath,
					Key:        primaryKey,
					TargetType: field.Type.String(),
					Err:        ErrMissingRequired,
				})
			}
			continue
		}

		// Strict decoding into field
		primaryKey := matchedKey
		if primaryKey == "" && len(candidates) > 0 {
			primaryKey = candidates[0]
		} else if primaryKey == "" {
			primaryKey = tagInfo.primaryKey
		}

		if err := d.decodeField(fieldVal, valStr, tagInfo); err != nil {
			valErr.Add(FieldError{
				Field:      fieldPath,
				Key:        primaryKey,
				Value:      valStr,
				TargetType: field.Type.String(),
				Err:        err,
			})
			continue
		}

		// Run field validator if implemented
		d.checkValidator(fieldVal.Addr(), fieldPath, valErr)
	}
}

type fieldTagInfo struct {
	primaryKey   string
	required     bool
	hasDefault   bool
	defaultValue string
	separator    string
	layout       string
}

func parseFieldTag(field reflect.StructField, prefix string) fieldTagInfo {
	info := fieldTagInfo{
		separator: ",",
	}

	// 1. Check env tag
	tag := field.Tag.Get("env")
	if tag == "" {
		tag = field.Tag.Get("config")
	}

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
	if def, ok := field.Tag.Lookup("default"); ok {
		info.hasDefault = true
		info.defaultValue = def
	} else if def, ok := field.Tag.Lookup("env-default"); ok {
		info.hasDefault = true
		info.defaultValue = def
	}

	// 3. Check explicit required tag
	if req, ok := field.Tag.Lookup("required"); ok {
		info.required = strings.EqualFold(req, "true") || req == "1"
	} else if req, ok := field.Tag.Lookup("env-required"); ok {
		info.required = strings.EqualFold(req, "true") || req == "1"
	}

	// 4. Separator tag for slices
	if sep := field.Tag.Get("sep"); sep != "" {
		info.separator = sep
	}

	// 5. Layout tag for time.Time
	if layout := field.Tag.Get("layout"); layout != "" {
		info.layout = layout
	}

	return info
}

func (d *Decoder) buildCandidateKeys(tagInfo fieldTagInfo, prefix, fieldName string) []string {
	var candidates []string

	if tagInfo.primaryKey != "" {
		// If key starts with /, treat as absolute (no prefix)
		if strings.HasPrefix(tagInfo.primaryKey, "/") {
			candidates = append(candidates, strings.TrimPrefix(tagInfo.primaryKey, "/"))
		} else {
			if prefix != "" {
				candidates = append(candidates, prefix+tagInfo.primaryKey)
			}
			candidates = append(candidates, tagInfo.primaryKey)
		}
	}

	// Default naming conventions
	snake := toScreamingSnake(fieldName)
	if prefix != "" {
		candidates = append(candidates, prefix+snake)
		candidates = append(candidates, prefix+fieldName)
	} else {
		candidates = append(candidates, snake)
		candidates = append(candidates, fieldName)
	}

	// Additional lower/dot forms
	lowerSnake := strings.ToLower(snake)
	if prefix != "" {
		lowerPrefix := strings.ToLower(strings.TrimRight(prefix, "_."))
		candidates = append(candidates, lowerPrefix+"."+lowerSnake)
		candidates = append(candidates, lowerPrefix+"_"+lowerSnake)
	} else {
		candidates = append(candidates, lowerSnake)
	}

	return candidates
}

func (d *Decoder) decodeField(v reflect.Value, raw string, tagInfo fieldTagInfo) error {
	// 1. Special types with custom logic: time.Duration, time.Time, *url.URL, net.IP
	if v.Type() == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", raw, err)
		}
		v.SetInt(int64(d))
		return nil
	}

	if v.Type() == reflect.TypeOf(time.Time{}) {
		return decodeTime(v, raw, tagInfo.layout)
	}

	if v.Type() == reflect.TypeOf(&url.URL{}) {
		parsedURL, err := url.Parse(raw)
		if err != nil {
			return fmt.Errorf("invalid URL %q: %w", raw, err)
		}
		v.Set(reflect.ValueOf(parsedURL))
		return nil
	}

	if v.Type() == reflect.TypeOf(net.IP{}) {
		ip := net.ParseIP(strings.TrimSpace(raw))
		if ip == nil {
			return fmt.Errorf("invalid IP address %q", raw)
		}
		v.Set(reflect.ValueOf(ip))
		return nil
	}

	// 2. TextUnmarshaler check (both value and pointer)
	if v.CanAddr() {
		if u, ok := v.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return u.UnmarshalText([]byte(raw))
		}
	}
	if u, ok := v.Interface().(encoding.TextUnmarshaler); ok {
		return u.UnmarshalText([]byte(raw))
	}

	// 3. BinaryUnmarshaler check
	if v.CanAddr() {
		if u, ok := v.Addr().Interface().(encoding.BinaryUnmarshaler); ok {
			return u.UnmarshalBinary([]byte(raw))
		}
	}

	// 4. Pointer types
	if v.Kind() == reflect.Pointer {
		elemType := v.Type().Elem()
		elemVal := reflect.New(elemType).Elem()
		if err := d.decodeField(elemVal, raw, tagInfo); err != nil {
			return err
		}
		v.Set(elemVal.Addr())
		return nil
	}

	// 5. Standard primitive kinds
	switch v.Kind() {
	case reflect.String:
		v.SetString(raw)
		return nil

	case reflect.Bool:
		b, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return fmt.Errorf("expected boolean (true/false/1/0), got %q: %w", raw, err)
		}
		v.SetBool(b)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(strings.TrimSpace(raw), 0, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("expected integer, got %q: %w", raw, err)
		}
		v.SetInt(i)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u, err := strconv.ParseUint(strings.TrimSpace(raw), 0, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("expected unsigned integer, got %q: %w", raw, err)
		}
		v.SetUint(u)
		return nil

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(strings.TrimSpace(raw), v.Type().Bits())
		if err != nil {
			return fmt.Errorf("expected floating-point number, got %q: %w", raw, err)
		}
		v.SetFloat(f)
		return nil

	case reflect.Slice:
		return d.decodeSlice(v, raw, tagInfo)

	case reflect.Map:
		return d.decodeMap(v, raw, tagInfo)

	default:
		return fmt.Errorf("unsupported target kind: %s", v.Kind())
	}
}

func (d *Decoder) decodeSlice(v reflect.Value, raw string, tagInfo fieldTagInfo) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		return nil
	}

	sep := tagInfo.separator
	if sep == "" {
		sep = ","
	}

	parts := strings.Split(raw, sep)
	slice := reflect.MakeSlice(v.Type(), len(parts), len(parts))

	for i, part := range parts {
		elem := slice.Index(i)
		token := strings.TrimSpace(part)
		if err := d.decodeField(elem, token, tagInfo); err != nil {
			return fmt.Errorf("element at index [%d]: %w", i, err)
		}
	}

	v.Set(slice)
	return nil
}

func (d *Decoder) decodeMap(v reflect.Value, raw string, tagInfo fieldTagInfo) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		v.Set(reflect.MakeMap(v.Type()))
		return nil
	}

	// Supported format: "k1=v1,k2=v2" or "k1:v1,k2:v2"
	sep := tagInfo.separator
	if sep == "" {
		sep = ","
	}

	pairs := strings.Split(raw, sep)
	mapVal := reflect.MakeMapWithSize(v.Type(), len(pairs))
	keyType := v.Type().Key()
	valType := v.Type().Elem()

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		var kStr, vStr string
		if idx := strings.IndexAny(pair, "=:"); idx >= 0 {
			kStr = strings.TrimSpace(pair[:idx])
			vStr = strings.TrimSpace(pair[idx+1:])
		} else {
			return fmt.Errorf("invalid map entry %q (expected key=value or key:value)", pair)
		}

		keyVal := reflect.New(keyType).Elem()
		if err := d.decodeField(keyVal, kStr, tagInfo); err != nil {
			return fmt.Errorf("map key %q: %w", kStr, err)
		}

		elemVal := reflect.New(valType).Elem()
		if err := d.decodeField(elemVal, vStr, tagInfo); err != nil {
			return fmt.Errorf("map value for key %q: %w", kStr, err)
		}

		mapVal.SetMapIndex(keyVal, elemVal)
	}

	v.Set(mapVal)
	return nil
}

func decodeTime(v reflect.Value, raw string, customLayout string) error {
	raw = strings.TrimSpace(raw)
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RubyDate,
		time.ANSIC,
	}

	if customLayout != "" {
		layouts = append([]string{customLayout}, layouts...)
	}

	for _, l := range layouts {
		if t, err := time.Parse(l, raw); err == nil {
			v.Set(reflect.ValueOf(t))
			return nil
		}
	}

	return fmt.Errorf("cannot parse %q as time.Time (expected RFC3339 or 'YYYY-MM-DD HH:MM:SS')", raw)
}

func (d *Decoder) checkValidator(v reflect.Value, fieldPath string, valErr *ValidationError) {
	if !v.IsValid() {
		return
	}

	if val, ok := v.Interface().(Validator); ok {
		if err := val.Validate(); err != nil {
			valErr.Add(FieldError{
				Field: fieldPath,
				Err:   fmt.Errorf("%w: %v", ErrValidationFailed, err),
			})
		}
	}
}

func isConfigStruct(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}

	// Special stdlib structs handled as scalar values
	if t == reflect.TypeOf(time.Time{}) || t == reflect.TypeOf(url.URL{}) || t == reflect.TypeOf(net.IP{}) {
		return false
	}

	// If type or *type implements TextUnmarshaler or BinaryUnmarshaler, it's a scalar value
	ptrType := reflect.PointerTo(t)
	textUnmarshaler := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	binaryUnmarshaler := reflect.TypeOf((*encoding.BinaryUnmarshaler)(nil)).Elem()

	if t.Implements(textUnmarshaler) || ptrType.Implements(textUnmarshaler) {
		return false
	}
	if t.Implements(binaryUnmarshaler) || ptrType.Implements(binaryUnmarshaler) {
		return false
	}

	return true
}

// toScreamingSnake converts camelCase / PascalCase to SCREAMING_SNAKE_CASE.
// e.g. "ServerPort" -> "SERVER_PORT", "HTTPTimeout" -> "HTTP_TIMEOUT".
func toScreamingSnake(s string) string {
	var sb strings.Builder
	runes := []rune(s)
	n := len(runes)

	for i := 0; i < n; i++ {
		r := runes[i]
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			// Add underscore if preceded by lowercase or if followed by lowercase (e.g. HTTPTimeout -> HTTP_TIMEOUT)
			if unicode.IsLower(prev) || unicode.IsDigit(prev) ||
				(i+1 < n && unicode.IsLower(runes[i+1])) {
				sb.WriteByte('_')
			}
		}
		sb.WriteRune(unicode.ToUpper(r))
	}

	return sb.String()
}
