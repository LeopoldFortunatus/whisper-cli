package fsx

import (
	"io"
	"os"
	"path/filepath"
)

type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

type FS interface {
	Abs(path string) (string, error)
	Stat(path string) (os.FileInfo, error)
	ReadDir(path string) ([]os.DirEntry, error)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Open(path string) (ReadSeekCloser, error)
}

type OS struct{}

func (OS) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

func (OS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (OS) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (OS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (OS) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (OS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (OS) Open(path string) (ReadSeekCloser, error) {
	return os.Open(path)
}
