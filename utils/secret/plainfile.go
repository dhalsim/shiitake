package secret

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func PlainFileDriver() *PlainFile {
	return &PlainFile{path: encryptedFilePath(context.Background())}
}

type PlainFile struct {
	path string // directory
}

func (s *PlainFile) Set(key string, value []byte) error {
	if err := os.WriteFile(filepath.Join(s.path, key), value, 0600); err != nil {
		return errors.Wrap(err, "failed to write value to file")
	}
	return nil
}

func (s *PlainFile) Get(key string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(s.path, key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, errors.Wrap(err, "failed to get key")
	}
	return b, nil
}
