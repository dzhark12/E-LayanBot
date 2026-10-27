package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var safeKey = regexp.MustCompile(`^[0-9a-f-]{36}\.(jpg|png)$`)

type Local struct{ root string }

func NewLocal(root string) (*Local, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0700); err != nil {
		return nil, err
	}
	return &Local{root: abs}, nil
}

func (s *Local) Create(key string, data []byte) error {
	if !safeKey.MatchString(key) {
		return errors.New("invalid storage key")
	}
	path := filepath.Join(s.root, key)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	return f.Close()
}

func (s *Local) Open(key string) (*os.File, error) {
	if !safeKey.MatchString(key) {
		return nil, errors.New("invalid storage key")
	}
	path := filepath.Join(s.root, key)
	return os.Open(path)
}

func (s *Local) Remove(key string) error {
	if !safeKey.MatchString(key) {
		return errors.New("invalid storage key")
	}
	path := filepath.Join(s.root, key)
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove attachment: %w", err)
	}
	return nil
}
