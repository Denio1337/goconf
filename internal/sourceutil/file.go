// Package sourceutil provides reusable utilities and base abstractions for configuration sources.
package sourceutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

// FileSource provides a thread-safe, reusable base implementation for file- and reader-based sources.
type FileSource struct {
	mu            sync.RWMutex
	path          string
	rawData       []byte
	readErr       error
	ignoreMissing bool
	namePrefix    string
}

// NewFileSource initializes a FileSource targeting a file on disk.
func NewFileSource(path, namePrefix string) FileSource {
	return FileSource{
		path:       path,
		namePrefix: namePrefix,
	}
}

// NewReaderSource initializes a FileSource targeting an io.Reader.
// The content is eagerly buffered to allow multiple re-reads without data loss.
// If reading fails, the error is preserved and returned on Load.
func NewReaderSource(r io.Reader, namePrefix string) FileSource {
	var raw []byte
	var err error
	if r != nil {
		raw, err = io.ReadAll(r)
	}
	return FileSource{
		rawData:    raw,
		readErr:    err,
		namePrefix: namePrefix,
	}
}

// SetIgnoreMissing safely configures whether missing files should be ignored.
func (f *FileSource) SetIgnoreMissing(ignore bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ignoreMissing = ignore
}

// IgnoreMissing safely returns whether missing files should be ignored.
func (f *FileSource) IgnoreMissing() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.ignoreMissing
}

// Name returns the descriptive name of the source.
func (f *FileSource) Name() string {
	if f.path != "" {
		return fmt.Sprintf("%s:%s", f.namePrefix, f.path)
	}
	return fmt.Sprintf("%s:reader", f.namePrefix)
}

// ReadAndParse loads the underlying data (from buffered bytes or disk) and invokes the parser function.
func (f *FileSource) ReadAndParse(ctx context.Context, parse func(r io.Reader) (map[string]any, error)) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if f.readErr != nil {
		return nil, fmt.Errorf("reading %s: %w", f.Name(), f.readErr)
	}

	var r io.Reader
	if f.rawData != nil {
		r = bytes.NewReader(f.rawData)
	} else {
		file, err := os.Open(f.path)
		if err != nil {
			if os.IsNotExist(err) && f.IgnoreMissing() {
				return make(map[string]any), nil
			}
			return nil, fmt.Errorf("failed to open %s file %q: %w", f.namePrefix, f.path, err)
		}
		defer file.Close()
		r = file
	}

	data, err := parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", f.Name(), err)
	}
	if data == nil {
		return make(map[string]any), nil
	}
	return data, nil
}
