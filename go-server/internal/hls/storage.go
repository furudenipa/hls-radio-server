package hls

import (
	"os"
	"path/filepath"
)

// PlaylistStorage defines the storage interface for playlists
type PlaylistStorage interface {
	Store(key string, content PlaylistContent) error
	Load(key string) (PlaylistContent, error)
}

// FileSystem defines the file system operations
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
}

// FileStorage implements PlaylistStorage using the file system
type FileStorage struct {
	fs        FileSystem
	directory string
}

func NewFileStorage(fs FileSystem, directory string) *FileStorage {
	return &FileStorage{
		fs:        fs,
		directory: directory,
	}
}

func (s *FileStorage) Store(key string, content PlaylistContent) error {
	path := filepath.Join(s.directory, key)
	return s.fs.WriteFile(path, content.Bytes(), 0644)
}

func (s *FileStorage) Load(key string) (PlaylistContent, error) {
	path := filepath.Join(s.directory, key)
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &DefaultPlaylistContent{data: data}, nil
}

// DefaultFileSystem implements FileSystem using os package
type DefaultFileSystem struct{}

func (fs DefaultFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs DefaultFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}
