// Package fs is archfit's adapter for filesystem writes. CLAUDE.md §3 (P5)
// requires all filesystem mutations to go through internal/adapter/ so they
// can be faked in tests. No other package may call os.WriteFile, os.Remove,
// os.MkdirAll, or os.OpenFile for writing.
//
// Read-only operations (os.ReadFile, os.Stat) are included for symmetry so
// callers that need both read and write don't import two packages.
package fs

import (
	"io/fs"
	"os"
	"time"
)

// FS abstracts the filesystem operations the fix engine needs. Implementations
// must be safe for sequential use from a single goroutine — no concurrent
// safety is required.
type FS interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Remove(name string) error
	Stat(name string) (fs.FileInfo, error)
	// OpenFile is used by the log writer for append operations.
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
}

// Real delegates to the os package. Use in production paths only.
type Real struct{}

// NewReal returns a Real filesystem adapter.
func NewReal() *Real { return &Real{} }

// ReadFile reads a file via os.ReadFile.
func (*Real) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }

// WriteFile writes a file via os.WriteFile.
func (*Real) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// MkdirAll creates directories via os.MkdirAll.
func (*Real) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }

// Remove deletes a file via os.Remove.
func (*Real) Remove(name string) error { return os.Remove(name) }

// Stat returns file info via os.Stat.
func (*Real) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }

// OpenFile opens a file via os.OpenFile.
func (*Real) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

// Memory is an in-memory filesystem for tests. Files are stored in a map
// keyed by absolute path. Directories are implicit (not tracked).
type Memory struct {
	Files map[string][]byte
}

// NewMemory returns an empty in-memory filesystem.
func NewMemory() *Memory {
	return &Memory{Files: map[string][]byte{}}
}

// ReadFile returns the file contents from the in-memory map.
func (m *Memory) ReadFile(name string) ([]byte, error) {
	data, ok := m.Files[name]
	if !ok {
		return nil, &os.PathError{Op: "read", Path: name, Err: os.ErrNotExist}
	}
	return append([]byte(nil), data...), nil // defensive copy
}

// WriteFile stores file contents in the in-memory map.
func (m *Memory) WriteFile(name string, data []byte, _ os.FileMode) error {
	m.Files[name] = append([]byte(nil), data...) // defensive copy
	return nil
}

// MkdirAll is a no-op for the in-memory filesystem.
func (m *Memory) MkdirAll(_ string, _ os.FileMode) error { return nil }

// Remove deletes a file from the in-memory map.
func (m *Memory) Remove(name string) error {
	if _, ok := m.Files[name]; !ok {
		return &os.PathError{Op: "remove", Path: name, Err: os.ErrNotExist}
	}
	delete(m.Files, name)
	return nil
}

// Stat returns file info for a file in the in-memory map.
func (m *Memory) Stat(name string) (fs.FileInfo, error) {
	if _, ok := m.Files[name]; !ok {
		return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
	}
	return memFileInfo{name: name, size: int64(len(m.Files[name]))}, nil
}

// OpenFile always returns an error for the in-memory filesystem.
// The fix log falls back to read-modify-write via ReadFile + WriteFile.
func (m *Memory) OpenFile(name string, _ int, _ os.FileMode) (*os.File, error) {
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrPermission}
}

type memFileInfo struct {
	name string
	size int64
}

// Name returns the file name.
func (fi memFileInfo) Name() string { return fi.name }

// Size returns the file size.
func (fi memFileInfo) Size() int64 { return fi.size }

// Mode returns a fixed file mode.
func (fi memFileInfo) Mode() os.FileMode { return 0o644 }

// IsDir always returns false.
func (fi memFileInfo) IsDir() bool { return false }

// Sys returns nil.
func (fi memFileInfo) Sys() any { return nil }

// ModTime returns the zero time.
func (fi memFileInfo) ModTime() time.Time { return time.Time{} }
