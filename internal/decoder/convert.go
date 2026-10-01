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
