package storage

import (
	"context"
	"os"
	"path/filepath"
)

// Storage interface for file storage operations
type Storage interface {
	// Upload uploads a local file to storage
	Upload(ctx context.Context, localPath, remotePath string) error

	// Download downloads a file from storage to local path
	Download(ctx context.Context, remotePath string) (localPath string, err error)

	// List lists files in a given path
	List(ctx context.Context, path string) ([]string, error)

	// ReadFile reads file contents
	ReadFile(ctx context.Context, path string) ([]byte, error)

	// WriteFile writes data to a file
	WriteFile(path string, data []byte) error

	// Delete deletes a file from storage
	Delete(ctx context.Context, path string) error

	// GetLocalPath returns the local filesystem path for a given storage path
	GetLocalPath(path string) string

	// EnsureDirectory ensures a directory exists
	EnsureDirectory(path string) error

	// Cleanup removes temporary local files
	Cleanup(localPath string) error
}

// LocalStorage implements Storage using local filesystem
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(basePath string) *LocalStorage {
	// Ensure base path exists
	os.MkdirAll(basePath, 0755)
	return &LocalStorage{
		basePath: basePath,
	}
}

// Upload for local storage is a no-op since files are already local
func (s *LocalStorage) Upload(ctx context.Context, localPath, remotePath string) error {
	// In local storage, we might copy to a specific directory structure
	destPath := filepath.Join(s.basePath, remotePath)
	if localPath != destPath {
		// Copy file if paths are different
		data, err := os.ReadFile(localPath)
		if err != nil {
			return err
		}
		return s.WriteFile(destPath, data)
	}
	return nil
}

// Download for local storage returns the actual path
func (s *LocalStorage) Download(ctx context.Context, remotePath string) (string, error) {
	localPath := filepath.Join(s.basePath, remotePath)
	if _, err := os.Stat(localPath); err != nil {
		return "", err
	}
	return localPath, nil
}

// List lists files in a directory
func (s *LocalStorage) List(ctx context.Context, path string) ([]string, error) {
	fullPath := filepath.Join(s.basePath, path)
	var files []string

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Return relative path from base
			relPath, _ := filepath.Rel(s.basePath, path)
			files = append(files, relPath)
		}
		return nil
	})

	return files, err
}

// ReadFile reads file contents
func (s *LocalStorage) ReadFile(ctx context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(s.basePath, path)
	return os.ReadFile(fullPath)
}

// WriteFile writes data to a file
func (s *LocalStorage) WriteFile(path string, data []byte) error {
	fullPath := filepath.Join(s.basePath, path)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, 0644)
}

// Delete deletes a file
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)
	return os.Remove(fullPath)
}

// GetLocalPath returns the full local path
func (s *LocalStorage) GetLocalPath(path string) string {
	return filepath.Join(s.basePath, path)
}

// EnsureDirectory ensures a directory exists
func (s *LocalStorage) EnsureDirectory(path string) error {
	fullPath := filepath.Join(s.basePath, path)
	return os.MkdirAll(fullPath, 0755)
}

// Cleanup removes temporary files
func (s *LocalStorage) Cleanup(localPath string) error {
	// Only clean up if it's a temporary file
	if filepath.HasPrefix(localPath, os.TempDir()) {
		return os.RemoveAll(localPath)
	}
	return nil
}
