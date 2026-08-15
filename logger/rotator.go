package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotatingFile is an io.WriteCloser that rotates its underlying file once it
// exceeds maxSize, and prunes old backups by count and by age.
type RotatingFile struct {
	mu         sync.Mutex
	filename   string
	maxSize    int64 // bytes
	maxBackups int
	maxAge     int // days
	size       int64
	file       *os.File
}

// NewRotatingFile opens (or creates) a log file with rotation. maxSizeMB is
// the maximum size in MB before rotating, maxBackups is how many old backups
// to keep, and maxAgeDays is the maximum retention in days (0 means no limit).
func NewRotatingFile(filename string, maxSizeMB, maxBackups, maxAgeDays int) (*RotatingFile, error) {
	rf := &RotatingFile{
		filename:   filename,
		maxSize:    int64(maxSizeMB) * 1024 * 1024,
		maxBackups: maxBackups,
		maxAge:     maxAgeDays,
	}
	if err := rf.open(); err != nil {
		return nil, err
	}
	return rf, nil
}

// open opens the log file in append mode, restoring the current size when the
// file already exists.
func (rf *RotatingFile) open() error {
	info, err := os.Stat(rf.filename)
	if err == nil {
		rf.size = info.Size()
	}
	f, err := os.OpenFile(rf.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	rf.file = f
	return nil
}

// Write appends p to the log file, rotating it first when the write would
// exceed maxSize.
func (rf *RotatingFile) Write(p []byte) (n int, err error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.size+int64(len(p)) > rf.maxSize && rf.maxSize > 0 {
		if err := rf.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = rf.file.Write(p)
	rf.size += int64(n)
	return n, err
}

// Sync flushes the log file to disk.
func (rf *RotatingFile) Sync() error {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.file != nil {
		return rf.file.Sync()
	}
	return nil
}

// Close closes the log file.
func (rf *RotatingFile) Close() error {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.file != nil {
		return rf.file.Close()
	}
	return nil
}

// rotate renames the current file to a timestamped backup, prunes old
// backups, and reopens a fresh log file.
func (rf *RotatingFile) rotate() error {
	if rf.file != nil {
		rf.file.Close()
	}

	backup := fmt.Sprintf("%s.%s", rf.filename, time.Now().Format("20060102-150405"))
	os.Rename(rf.filename, backup)

	rf.cleanup()
	return rf.open()
}

// cleanup removes backups that are older than maxAge days or exceed
// maxBackups, keeping the most recent ones.
func (rf *RotatingFile) cleanup() {
	dir := filepath.Dir(rf.filename)
	base := filepath.Base(rf.filename)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var backups []os.DirEntry
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), base+".") {
			backups = append(backups, e)
		}
	}

	// Prune by age.
	if rf.maxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -rf.maxAge)
		for _, e := range backups {
			info, _ := e.Info()
			if info != nil && info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(dir, e.Name()))
			}
		}
	}

	// Prune by count, keeping the most recent backups.
	if rf.maxBackups > 0 {
		sort.Slice(backups, func(i, j int) bool {
			ii, _ := backups[i].Info()
			ji, _ := backups[j].Info()
			return ii.ModTime().After(ji.ModTime())
		})
		for i := rf.maxBackups; i < len(backups); i++ {
			os.Remove(filepath.Join(dir, backups[i].Name()))
		}
	}
}

var _ io.WriteCloser = (*RotatingFile)(nil)
