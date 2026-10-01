package goconf

import (
	"encoding"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Validator is an optional interface that structs or fields can implement
// to execute custom business-level validation logic after decoding.
type Validator interface {
	Validate() error
}

// Decoder decodes configuration from a Store into a target struct with strict schema validation.
type Decoder struct {
	store         *Store
	prefix        string
	strictUnknown bool
	consumedKeys  map[string]bool
}

// NewDecoder creates a new Decoder configured with the provided Store.
func NewDecoder(store *Store) *Decoder {
	return &Decoder{
		store:        store,
		consumedKeys: make(map[string]bool),
	}
}

// SetPrefix sets a global key prefix to be prepended to all struct fields.
func (d *Decoder) SetPrefix(prefix string) {
	d.prefix = prefix
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
	d.decodeStruct(elem, d.prefix, "", valErr)

	// Check if the root struct itself implements Validator
	d.checkValidator(val, "", false, valErr)

	// If strict unknown keys enabled, check for unused keys
	if d.strictUnknown {
		for key := range d.store.StrictKeys() {
			lower := strings.ToLower(key)
			if !d.consumedKeys[key] && !d.consumedKeys[lower] &&
				!d.consumedKeys[strings.ReplaceAll(lower, ".", "_")] &&
				!d.consumedKeys[strings.ReplaceAll(lower, ".", "__")] &&
				!d.consumedKeys[strings.ReplaceAll(lower, "_", ".")] {
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

			// Check prefix tag: `prefix:"..."`
			if structPrefix, ok := field.Tag.Lookup(TagPrefix); ok {
				childPrefix = prefix + structPrefix
			} else if keyTag, ok := field.Tag.Lookup(TagKey); ok && keyTag != "" {
				keyParts := strings.Split(keyTag, ",")
				baseKey := strings.TrimSpace(keyParts[0])
				if baseKey != "" {
					if !strings.HasSuffix(baseKey, "_") && !strings.HasSuffix(baseKey, ".") {
						baseKey += "_"
					}
					childPrefix = prefix + baseKey
				}
			} else if !field.Anonymous {
				childPrefix = prefix + toScreamingSnake(field.Name) + "_"
			}

			// Mark parent container / prefix keys as consumed so strictUnknown doesn't flag them
			cleanPrefix := strings.TrimRight(childPrefix, "_.")
			d.markConsumed(cleanPrefix)
			d.markConsumed(field.Name)

			if field.Type.Kind() == reflect.Pointer {
				if fieldVal.IsNil() {
					fieldVal.Set(reflect.New(field.Type.Elem()))
				}
				d.decodeStruct(fieldVal.Elem(), childPrefix, fieldPath, valErr)
				d.checkValidator(fieldVal, fieldPath, false, valErr)
			} else {
				d.decodeStruct(fieldVal, childPrefix, fieldPath, valErr)
				d.checkValidator(fieldVal.Addr(), fieldPath, false, valErr)
			}
			continue
		}

		// Parse tags and options
		tagInfo := parseFieldTag(field)

		// Candidate keys for lookup in the store
		candidates := d.buildCandidateKeys(tagInfo, prefix, field.Name)

		// Look up value in store
		rawVal, matchedKey, found := d.store.Get(candidates...)
		if found {
			d.markConsumed(matchedKey)
			for _, c := range candidates {
				d.markConsumed(c)
			}
		}

		// Check for missing value
		var rawValue any
		hasValue := false

		if found && rawVal != nil {
			rawValue = rawVal
			switch rv := rawVal.(type) {
			case string:
				hasValue = strings.TrimSpace(rv) != ""
			default:
				hasValue = true
			}
		}

		// Fallback to default if empty or not found
		if !hasValue && tagInfo.hasDefault {
			rawValue = tagInfo.defaultValue
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
					IsSecret:   tagInfo.isSecret,
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

		if err := d.decodeFieldValue(fieldVal, rawValue, tagInfo); err != nil {
			valErr.Add(FieldError{
				Field:      fieldPath,
				Key:        primaryKey,
				Value:      rawValue,
				TargetType: field.Type.String(),
				Err:        err,
				IsSecret:   tagInfo.isSecret,
			})
			continue
		}

		// Run field validator if implemented
		d.checkValidator(fieldVal.Addr(), fieldPath, tagInfo.isSecret, valErr)
	}
}

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

	// 6. Secret tag or Secret[T] wrapper
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
		}
		// Also allow key without prefix as fallback
		candidates = append(candidates, tagInfo.primaryKey)
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

func (d *Decoder) markConsumed(key string) {
	if key == "" {
		return
	}
	d.consumedKeys[key] = true
	d.consumedKeys[strings.ToLower(key)] = true
	if strings.Contains(key, ".") {
		d.consumedKeys[strings.ToLower(strings.ReplaceAll(key, ".", "_"))] = true
		d.consumedKeys[strings.ToLower(strings.ReplaceAll(key, ".", "__"))] = true
	} else if strings.Contains(key, "_") {
		d.consumedKeys[strings.ToLower(strings.ReplaceAll(key, "_", "."))] = true
	}
}

func (d *Decoder) decodeFieldValue(v reflect.Value, raw any, tagInfo fieldTagInfo) error {
	if raw == nil {
		return nil
	}

	// Secret[T] wrapper support
	markerType := reflect.TypeFor[secretMarker]()
	if v.Type().Implements(markerType) || (v.CanAddr() && v.Addr().Type().Implements(markerType)) {
		targetVal := v
		if targetVal.Kind() == reflect.Pointer {
			if targetVal.IsNil() {
				targetVal.Set(reflect.New(targetVal.Type().Elem()))
			}
			targetVal = targetVal.Elem()
		}
		innerType := targetVal.Type().Field(0).Type
		innerVal := reflect.New(innerType).Elem()
		if err := d.decodeFieldValue(innerVal, raw, tagInfo); err != nil {
			return err
		}
		if targetVal.CanAddr() {
			method := targetVal.Addr().MethodByName("Set")
			if method.IsValid() {
				method.Call([]reflect.Value{innerVal})
			}
		}
		return nil
	}

	// Special types that shouldn't be handled as generic slices or maps:
	// net.IP is defined as []byte in the stdlib, but should be decoded as an IP string
	if v.Type() == reflect.TypeOf(net.IP{}) {
		rawStr, ok := raw.(string)
		if !ok {
			rawStr = fmt.Sprint(raw)
		}
		return d.decodeField(v, rawStr, tagInfo)
	}

	// Types implementing TextUnmarshaler or BinaryUnmarshaler
	textUnmarshaler := reflect.TypeFor[encoding.TextUnmarshaler]()
	binaryUnmarshaler := reflect.TypeFor[encoding.BinaryUnmarshaler]()
	if v.Type().Implements(textUnmarshaler) || (v.CanAddr() && v.Addr().Type().Implements(textUnmarshaler)) ||
		v.Type().Implements(binaryUnmarshaler) || (v.CanAddr() && v.Addr().Type().Implements(binaryUnmarshaler)) {
		rawStr, ok := raw.(string)
		if !ok {
			rawStr = fmt.Sprint(raw)
		}
		return d.decodeField(v, rawStr, tagInfo)
	}

	// 1. If target is slice
	if v.Kind() == reflect.Slice {
		rawVal := reflect.ValueOf(raw)
		if rawVal.Kind() == reflect.Slice {
			slice := reflect.MakeSlice(v.Type(), rawVal.Len(), rawVal.Len())
			for i := 0; i < rawVal.Len(); i++ {
				elem := slice.Index(i)
				item := rawVal.Index(i).Interface()
				if err := d.decodeFieldValue(elem, item, tagInfo); err != nil {
					return fmt.Errorf("element at index [%d]: %w", i, err)
				}
			}
			v.Set(slice)
			return nil
		}
		rawStr, ok := raw.(string)
		if !ok {
			rawStr = fmt.Sprint(raw)
		}
		return d.decodeSlice(v, rawStr, tagInfo)
	}

	// 2. If target is map
	if v.Kind() == reflect.Map {
		rawVal := reflect.ValueOf(raw)
		if rawVal.Kind() == reflect.Map {
			mapVal := reflect.MakeMapWithSize(v.Type(), rawVal.Len())
			keyType := v.Type().Key()
			valType := v.Type().Elem()
			for _, k := range rawVal.MapKeys() {
				newKey := reflect.New(keyType).Elem()
				newElem := reflect.New(valType).Elem()

				if err := d.decodeFieldValue(newKey, k.Interface(), tagInfo); err != nil {
					return fmt.Errorf("map key %v: %w", k, err)
				}
				if err := d.decodeFieldValue(newElem, rawVal.MapIndex(k).Interface(), tagInfo); err != nil {
					return fmt.Errorf("map key %v value: %w", k, err)
				}
				mapVal.SetMapIndex(newKey, newElem)
			}
			v.Set(mapVal)
			return nil
		}
		rawStr, ok := raw.(string)
		if !ok {
			rawStr = fmt.Sprint(raw)
		}
		return d.decodeMap(v, rawStr, tagInfo)
	}

	// 3. Pointer types
	if v.Kind() == reflect.Pointer {
		elemType := v.Type().Elem()
		elemVal := reflect.New(elemType).Elem()
		if err := d.decodeFieldValue(elemVal, raw, tagInfo); err != nil {
			return err
		}
		v.Set(elemVal.Addr())
		return nil
	}

	// 4. Direct assignment if raw value is already assignable to target type
	rawVal := reflect.ValueOf(raw)
	if rawVal.IsValid() && rawVal.Type().AssignableTo(v.Type()) {
		v.Set(rawVal)
		return nil
	}

	// 5. Fast typed conversions for numeric, boolean, and time values without string roundtrip
	if err, handled := tryDirectTypeConversion(v, raw); handled {
		return err
	}

	// 6. String-based fallback (for .env, OS env vars, and custom string unmarshalers)
	rawStr, ok := raw.(string)
	if !ok {
		rawStr = fmt.Sprint(raw)
	}
	return d.decodeField(v, rawStr, tagInfo)
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

	// Support array brackets like `["a", "b"]` or `[a, b]`
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		raw = strings.TrimSpace(raw[1 : len(raw)-1])
		if raw == "" {
			v.Set(reflect.MakeSlice(v.Type(), 0, 0))
			return nil
		}
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
		if len(token) >= 2 && ((token[0] == '"' && token[len(token)-1] == '"') || (token[0] == '\'' && token[len(token)-1] == '\'')) {
			token = token[1 : len(token)-1]
		}
		if err := d.decodeFieldValue(elem, token, tagInfo); err != nil {
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
		if err := d.decodeFieldValue(keyVal, kStr, tagInfo); err != nil {
			return fmt.Errorf("map key %q: %w", kStr, err)
		}

		elemVal := reflect.New(valType).Elem()
		if err := d.decodeFieldValue(elemVal, vStr, tagInfo); err != nil {
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

func (d *Decoder) checkValidator(v reflect.Value, fieldPath string, isSecret bool, valErr *ValidationError) {
	if !v.IsValid() {
		return
	}

	if val, ok := v.Interface().(Validator); ok {
		if err := val.Validate(); err != nil {
			valErr.Add(FieldError{
				Field:    fieldPath,
				Err:      fmt.Errorf("%w: %v", ErrValidationFailed, err),
				IsSecret: isSecret,
			})
		}
	}
}

func isSecretType(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	markerType := reflect.TypeFor[secretMarker]()
	return t.Implements(markerType) || reflect.PointerTo(t).Implements(markerType)
}

func isConfigStruct(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}

	// Secret[T] is a wrapped scalar value, not a nested config struct
	if isSecretType(t) {
		return false
	}

	// Special stdlib structs handled as scalar values
	if t == reflect.TypeFor[time.Time]() || t == reflect.TypeFor[url.URL]() || t == reflect.TypeFor[net.IP]() {
		return false
	}

	// If type or *type implements TextUnmarshaler or BinaryUnmarshaler, it's a scalar value
	ptrType := reflect.PointerTo(t)
	textUnmarshaler := reflect.TypeFor[encoding.TextUnmarshaler]()
	binaryUnmarshaler := reflect.TypeFor[encoding.BinaryUnmarshaler]()

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

	for i := range n {
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

func tryDirectTypeConversion(v reflect.Value, raw any) (error, bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Type() == reflect.TypeFor[time.Duration]() {
			if d, ok := raw.(time.Duration); ok {
				v.SetInt(int64(d))
				return nil, true
			}
			return nil, false
		}

		if num, ok := raw.(json.Number); ok {
			i, err := num.Int64()
			if err != nil {
				return fmt.Errorf("expected integer, got %q: %w", num.String(), err), true
			}
			if v.OverflowInt(i) {
				return fmt.Errorf("integer overflow for %s: %d", v.Type(), i), true
			}
			v.SetInt(i)
			return nil, true
		}

		switch r := raw.(type) {
		case int:
			i := int64(r)
			if v.OverflowInt(i) {
				return fmt.Errorf("integer overflow for %s: %d", v.Type(), i), true
			}
			v.SetInt(i)
			return nil, true
		case int64:
			if v.OverflowInt(r) {
				return fmt.Errorf("integer overflow for %s: %d", v.Type(), r), true
			}
			v.SetInt(r)
			return nil, true
		case int32:
			v.SetInt(int64(r))
			return nil, true
		case int16:
			v.SetInt(int64(r))
			return nil, true
		case int8:
			v.SetInt(int64(r))
			return nil, true
		case uint:
			if uint64(r) > uint64(math.MaxInt64) || v.OverflowInt(int64(r)) {
				return fmt.Errorf("integer overflow for %s: %d", v.Type(), r), true
			}
			v.SetInt(int64(r))
			return nil, true
		case uint64:
			if r > uint64(math.MaxInt64) || v.OverflowInt(int64(r)) {
				return fmt.Errorf("integer overflow for %s: %d", v.Type(), r), true
			}
			v.SetInt(int64(r))
			return nil, true
		case uint32:
			v.SetInt(int64(r))
			return nil, true
		case uint16:
			v.SetInt(int64(r))
			return nil, true
		case uint8:
			v.SetInt(int64(r))
			return nil, true
		case float64:
			if r == math.Trunc(r) {
				i := int64(r)
				if v.OverflowInt(i) {
					return fmt.Errorf("integer overflow for %s: %d", v.Type(), i), true
				}
				v.SetInt(i)
				return nil, true
			}
		case float32:
			if float64(r) == math.Trunc(float64(r)) {
				i := int64(r)
				if v.OverflowInt(i) {
					return fmt.Errorf("integer overflow for %s: %d", v.Type(), i), true
				}
				v.SetInt(i)
				return nil, true
			}
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if num, ok := raw.(json.Number); ok {
			u, err := strconv.ParseUint(num.String(), 10, v.Type().Bits())
			if err != nil {
				return fmt.Errorf("expected unsigned integer, got %q: %w", num.String(), err), true
			}
			v.SetUint(u)
			return nil, true
		}

		switch r := raw.(type) {
		case uint:
			u := uint64(r)
			if v.OverflowUint(u) {
				return fmt.Errorf("unsigned integer overflow for %s: %d", v.Type(), u), true
			}
			v.SetUint(u)
			return nil, true
		case uint64:
			if v.OverflowUint(r) {
				return fmt.Errorf("unsigned integer overflow for %s: %d", v.Type(), r), true
			}
			v.SetUint(r)
			return nil, true
		case uint32:
			v.SetUint(uint64(r))
			return nil, true
		case uint16:
			v.SetUint(uint64(r))
			return nil, true
		case uint8:
			v.SetUint(uint64(r))
			return nil, true
		case int:
			if r < 0 || v.OverflowUint(uint64(r)) {
				return fmt.Errorf("cannot convert negative integer %d to %s", r, v.Type()), true
			}
			v.SetUint(uint64(r))
			return nil, true
		case int64:
			if r < 0 || v.OverflowUint(uint64(r)) {
				return fmt.Errorf("cannot convert negative integer %d to %s", r, v.Type()), true
			}
			v.SetUint(uint64(r))
			return nil, true
		case int32:
			if r < 0 {
				return fmt.Errorf("cannot convert negative integer %d to %s", r, v.Type()), true
			}
			v.SetUint(uint64(r))
			return nil, true
		case int16:
			if r < 0 {
				return fmt.Errorf("cannot convert negative integer %d to %s", r, v.Type()), true
			}
			v.SetUint(uint64(r))
			return nil, true
		case int8:
			if r < 0 {
				return fmt.Errorf("cannot convert negative integer %d to %s", r, v.Type()), true
			}
			v.SetUint(uint64(r))
			return nil, true
		case float64:
			if r >= 0 && r == math.Trunc(r) {
				u := uint64(r)
				if v.OverflowUint(u) {
					return fmt.Errorf("unsigned integer overflow for %s: %d", v.Type(), u), true
				}
				v.SetUint(u)
				return nil, true
			}
		case float32:
			if r >= 0 && float64(r) == math.Trunc(float64(r)) {
				u := uint64(r)
				if v.OverflowUint(u) {
					return fmt.Errorf("unsigned integer overflow for %s: %d", v.Type(), u), true
				}
				v.SetUint(u)
				return nil, true
			}
		}

	case reflect.Float32, reflect.Float64:
		if num, ok := raw.(json.Number); ok {
			f, err := num.Float64()
			if err != nil {
				return fmt.Errorf("expected float, got %q: %w", num.String(), err), true
			}
			if v.OverflowFloat(f) {
				return fmt.Errorf("float overflow for %s: %v", v.Type(), f), true
			}
			v.SetFloat(f)
			return nil, true
		}

		switch r := raw.(type) {
		case float64:
			if v.OverflowFloat(r) {
				return fmt.Errorf("float overflow for %s: %v", v.Type(), r), true
			}
			v.SetFloat(r)
			return nil, true
		case float32:
			v.SetFloat(float64(r))
			return nil, true
		case int:
			v.SetFloat(float64(r))
			return nil, true
		case int64:
			v.SetFloat(float64(r))
			return nil, true
		case int32:
			v.SetFloat(float64(r))
			return nil, true
		case uint:
			v.SetFloat(float64(r))
			return nil, true
		case uint64:
			v.SetFloat(float64(r))
			return nil, true
		case uint32:
			v.SetFloat(float64(r))
			return nil, true
		}

	case reflect.Bool:
		if b, ok := raw.(bool); ok {
			v.SetBool(b)
			return nil, true
		}

	case reflect.String:
		if s, ok := raw.(string); ok {
			v.SetString(s)
			return nil, true
		}
	}

	if v.Type() == reflect.TypeFor[time.Time]() {
		if t, ok := raw.(time.Time); ok {
			v.Set(reflect.ValueOf(t))
			return nil, true
		}
	}

	return nil, false
}
