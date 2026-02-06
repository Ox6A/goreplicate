package files

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type FileIndex struct {
	db *sql.DB
}

type FileEntry struct {
	Path    string    `json:"path"`
	Hash    string    `json:"hash"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	IsDir   bool      `json:"is_dir"`
}

func NewFileIndex(dbPath string) (*FileIndex, error) {
	db, err := sql.Open("sqlite3", "fileindex.db")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS files (
		path TEXT PRIMARY KEY,
		hash TEXT,
		size INTEGER,
		mod_time INTEGER,
		is_dir BOOLEAN
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_hash ON files (hash)`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &FileIndex{db: db}, nil
}

func (fi *FileIndex) getEntry(path string) (*FileEntry, error) {
	var entry FileEntry
	var modTime int64
	err := fi.db.QueryRow("SELECT path, hash, size, mod_time, is_dir FROM files WHERE path = ?", path).Scan(&entry.Path, &entry.Hash, &entry.Size, &modTime, &entry.IsDir)
	if err != nil {
		return nil, err
	}
	entry.ModTime = time.Unix(modTime, 0)
	return &entry, nil
}

func (fi *FileIndex) insertOrUpdateEntry(entry FileEntry) error {
	_, err := fi.db.Exec("INSERT OR REPLACE INTO files (path, hash, size, mod_time, is_dir) VALUES (?, ?, ?, ?, ?)", entry.Path, entry.Hash, entry.Size, entry.ModTime.Unix(), entry.IsDir)
	return err
}

func (fi *FileIndex) Close() error {
	return fi.db.Close()
}

func computeHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func (fi *FileIndex) IndexDirectory(rootPath string) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		entry := FileEntry{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		}

		if !info.IsDir() {
			hash, err := computeHash(path)
			if err != nil {
				return err
			}
			entry.Hash = hash
		}
		return fi.insertOrUpdateEntry(entry)
	})
}

// GetAllFiles returns all files in the index
func (fi *FileIndex) GetAllFiles() ([]FileEntry, error) {
	rows, err := fi.db.Query("SELECT path, hash, size, mod_time, is_dir FROM files ORDER BY path")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileEntry
	for rows.Next() {
		var entry FileEntry
		var modTime int64
		err := rows.Scan(&entry.Path, &entry.Hash, &entry.Size, &modTime, &entry.IsDir)
		if err != nil {
			return nil, err
		}
		entry.ModTime = time.Unix(modTime, 0)
		files = append(files, entry)
	}
	return files, rows.Err()
}
