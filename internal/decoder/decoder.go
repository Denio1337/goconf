// Package decoder provides configuration decoding from a Store into Go structs,
// handling reflection, type conversion, strict unknown key validation, and secret masking.
package decoder

import (
	"encoding"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Denio1337/goconf/internal/store"
)

// Decoder decodes configuration from a Store into a target struct with strict schema validation.
type Decoder struct {
	store         *store.Store
	prefix        string
	strictUnknown bool
	consumedKeys  map[string]bool
}

// SecretMarker is implemented by Secret[T] to indicate a sensitive value.
type SecretMarker interface {
	IsSecret()
}

// Validator is an optional interface that structs or fields can implement
// to execute custom business-level validation logic after decoding.
type Validator interface {
	Validate() error
}

// New creates a new Decoder configured with the provided Store.
func New(st *store.Store) *Decoder {
	return &Decoder{
		store:        st,
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

	// SecretMarker wrapper support
	markerType := reflect.TypeFor[SecretMarker]()
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
