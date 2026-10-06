package decoder

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
)

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

	parts := splitSliceElements(raw, sep)
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

// splitSliceElements splits raw by sep, respecting quotes and escapes.
func splitSliceElements(raw, sep string) []string {
	if sep == "" {
		sep = ","
	}
	var parts []string
	var current strings.Builder
	inDouble := false
	inSingle := false

	sepLen := len(sep)
	rawLen := len(raw)

	for i := 0; i < rawLen; i++ {
		c := raw[i]

		if c == '\\' && i+1 < rawLen {
			current.WriteByte(c)
			i++
			current.WriteByte(raw[i])
			continue
		}

		if c == '"' && !inSingle {
			inDouble = !inDouble
			current.WriteByte(c)
			continue
		}

		if c == '\'' && !inDouble {
			inSingle = !inSingle
			current.WriteByte(c)
			continue
		}

		if !inDouble && !inSingle && i+sepLen <= rawLen && raw[i:i+sepLen] == sep {
			parts = append(parts, current.String())
			current.Reset()
			i += sepLen - 1
			continue
		}

		current.WriteByte(c)
	}

	parts = append(parts, current.String())
	return parts
}

func (d *Decoder) decodeMap(v reflect.Value, raw string, tagInfo fieldTagInfo) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
		return nil
	}

	// Supported format: "k1=v1,k2=v2" or "k1:v1,k2:v2"
	sep := tagInfo.separator
	if sep == "" {
		sep = ","
	}

	pairs := splitSliceElements(raw, sep)
	mapVal := v
	if mapVal.IsNil() {
		mapVal = reflect.MakeMapWithSize(v.Type(), len(pairs))
	}
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
		existing := mapVal.MapIndex(keyVal)
		if existing.IsValid() {
			elemVal.Set(existing)
		}
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

func isSecretType(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	markerType := reflect.TypeFor[SecretMarker]()
	return t.Implements(markerType) || reflect.PointerTo(t).Implements(markerType)
}

func isConfigStruct(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}

	// SecretMarker is a wrapped scalar value, not a nested config struct
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

func toInt64(raw any) (int64, error, bool) {
	if num, ok := raw.(json.Number); ok {
		i, err := num.Int64()
		if err == nil {
			return i, nil, true
		}
		f, fErr := num.Float64()
		if fErr == nil {
			if f < float64(math.MinInt64) || f > float64(math.MaxInt64) {
				return 0, fmt.Errorf("%w: integer overflow: %v", ErrTypeMismatch, f), true
			}
			if f == math.Trunc(f) {
				return int64(f), nil, true
			}
		}
		return 0, fmt.Errorf("%w: expected integer, got %q: %v", ErrTypeMismatch, num.String(), err), true
	}

	switch r := raw.(type) {
	case int:
		return int64(r), nil, true
	case int64:
		return r, nil, true
	case int32:
		return int64(r), nil, true
	case int16:
		return int64(r), nil, true
	case int8:
		return int64(r), nil, true
	case uint:
		if uint64(r) > uint64(math.MaxInt64) {
			return 0, fmt.Errorf("%w: integer overflow: %d", ErrTypeMismatch, r), true
		}
		return int64(r), nil, true
	case uint64:
		if r > uint64(math.MaxInt64) {
			return 0, fmt.Errorf("%w: integer overflow: %d", ErrTypeMismatch, r), true
		}
		return int64(r), nil, true
	case uint32:
		return int64(r), nil, true
	case uint16:
		return int64(r), nil, true
	case uint8:
		return int64(r), nil, true
	case float64:
		if r == math.Trunc(r) {
			if r < float64(math.MinInt64) || r > float64(math.MaxInt64) {
				return 0, fmt.Errorf("%w: integer overflow: %v", ErrTypeMismatch, r), true
			}
			return int64(r), nil, true
		}
	case float32:
		f := float64(r)
		if f == math.Trunc(f) {
			if f < float64(math.MinInt64) || f > float64(math.MaxInt64) {
				return 0, fmt.Errorf("%w: integer overflow: %v", ErrTypeMismatch, r), true
			}
			return int64(r), nil, true
		}
	}

	return 0, nil, false
}

func toUint64(raw any) (uint64, error, bool) {
	if num, ok := raw.(json.Number); ok {
		u, err := strconv.ParseUint(num.String(), 10, 64)
		if err == nil {
			return u, nil, true
		}
		f, fErr := num.Float64()
		if fErr == nil {
			if f < 0 || f > float64(math.MaxUint64) {
				return 0, fmt.Errorf("%w: unsigned integer overflow: %v", ErrTypeMismatch, f), true
			}
			if f == math.Trunc(f) {
				return uint64(f), nil, true
			}
		}
		return 0, fmt.Errorf("%w: expected unsigned integer, got %q: %v", ErrTypeMismatch, num.String(), err), true
	}

	switch r := raw.(type) {
	case uint:
		return uint64(r), nil, true
	case uint64:
		return r, nil, true
	case uint32:
		return uint64(r), nil, true
	case uint16:
		return uint64(r), nil, true
	case uint8:
		return uint64(r), nil, true
	case int:
		if r < 0 {
			return 0, fmt.Errorf("%w: cannot convert negative integer %d to unsigned integer", ErrTypeMismatch, r), true
		}
		return uint64(r), nil, true
	case int64:
		if r < 0 {
			return 0, fmt.Errorf("%w: cannot convert negative integer %d to unsigned integer", ErrTypeMismatch, r), true
		}
		return uint64(r), nil, true
	case int32:
		if r < 0 {
			return 0, fmt.Errorf("%w: cannot convert negative integer %d to unsigned integer", ErrTypeMismatch, r), true
		}
		return uint64(r), nil, true
	case int16:
		if r < 0 {
			return 0, fmt.Errorf("%w: cannot convert negative integer %d to unsigned integer", ErrTypeMismatch, r), true
		}
		return uint64(r), nil, true
	case int8:
		if r < 0 {
			return 0, fmt.Errorf("%w: cannot convert negative integer %d to unsigned integer", ErrTypeMismatch, r), true
		}
		return uint64(r), nil, true
	case float64:
		if r >= 0 && r == math.Trunc(r) {
			if r > float64(math.MaxUint64) {
				return 0, fmt.Errorf("%w: unsigned integer overflow: %v", ErrTypeMismatch, r), true
			}
			return uint64(r), nil, true
		}
	case float32:
		f := float64(r)
		if f >= 0 && f == math.Trunc(f) {
			if f > float64(math.MaxUint64) {
				return 0, fmt.Errorf("%w: unsigned integer overflow: %v", ErrTypeMismatch, r), true
			}
			return uint64(r), nil, true
		}
	}

	return 0, nil, false
}

func toFloat64(raw any) (float64, error, bool) {
	if num, ok := raw.(json.Number); ok {
		f, err := num.Float64()
		if err != nil {
			return 0, fmt.Errorf("%w: expected float, got %q: %v", ErrTypeMismatch, num.String(), err), true
		}
		return f, nil, true
	}

	switch r := raw.(type) {
	case float64:
		return r, nil, true
	case float32:
		return float64(r), nil, true
	case int:
		return float64(r), nil, true
	case int64:
		return float64(r), nil, true
	case int32:
		return float64(r), nil, true
	case int16:
		return float64(r), nil, true
	case int8:
		return float64(r), nil, true
	case uint:
		return float64(r), nil, true
	case uint64:
		return float64(r), nil, true
	case uint32:
		return float64(r), nil, true
	case uint16:
		return float64(r), nil, true
	case uint8:
		return float64(r), nil, true
	}

	return 0, nil, false
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

		i, err, handled := toInt64(raw)
		if !handled {
			return nil, false
		}
		if err != nil {
			return err, true
		}
		if v.OverflowInt(i) {
			return fmt.Errorf("%w: integer overflow for %s: %d", ErrTypeMismatch, v.Type(), i), true
		}
		v.SetInt(i)
		return nil, true

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u, err, handled := toUint64(raw)
		if !handled {
			return nil, false
		}
		if err != nil {
			return err, true
		}
		if v.OverflowUint(u) {
			return fmt.Errorf("%w: unsigned integer overflow for %s: %d", ErrTypeMismatch, v.Type(), u), true
		}
		v.SetUint(u)
		return nil, true

	case reflect.Float32, reflect.Float64:
		f, err, handled := toFloat64(raw)
		if !handled {
			return nil, false
		}
		if err != nil {
			return err, true
		}
		if v.OverflowFloat(f) {
			return fmt.Errorf("%w: float overflow for %s: %v", ErrTypeMismatch, v.Type(), f), true
		}
		v.SetFloat(f)
		return nil, true

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
