package batterycontrol

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

type JournalStore interface {
	Load() (Record, error)
	Save(Record) error
}

// FileJournal is append-only: incomplete writes block control rather than
// guessing whether a command was sent. It contains no credentials.
type FileJournal struct {
	Path string
	lock *os.File
}

// OpenFileJournal holds a process-lifetime OS lock. A second evcc instance
// sharing this journal cannot claim the first instance's actions as its own.
func OpenFileJournal(path string) (*FileJournal, error) {
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := lockJournal(lock); err != nil {
		lock.Close()
		return nil, fmt.Errorf("journal already in use or lock unavailable: %w", err)
	}
	return &FileJournal{Path: path, lock: lock}, nil
}

func (f *FileJournal) Close() error {
	if f.lock != nil {
		return f.lock.Close()
	}
	return nil
}

func (f FileJournal) Load() (Record, error) {
	var last Record
	r, err := os.Open(f.Path)
	if errors.Is(err, os.ErrNotExist) {
		return last, nil
	}
	if err != nil {
		return last, err
	}
	defer r.Close()
	scan := bufio.NewReader(r)
	for {
		line, err := scan.ReadBytes('\n')
		if len(line) == 0 && errors.Is(err, io.EOF) {
			return last, nil
		}
		if err != nil {
			return last, fmt.Errorf("incomplete journal: %w", err)
		}
		var next Record
		if err := json.Unmarshal(line, &next); err != nil {
			return last, err
		}
		if next.Schema != 1 || next.Revision == "" {
			return last, fmt.Errorf("unsupported journal record")
		}
		last = next
	}
}

func (f FileJournal) Save(r Record) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	// Parent is the existing database directory, not an ephemeral temp folder.
	file, err := os.OpenFile(f.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(data, '\n'))
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	// Linux production requires the directory entry to be durable too.
	// Windows test hosts do not provide directory fsync through os.File.
	if runtime.GOOS != "windows" {
		dir, err := os.Open(filepath.Dir(f.Path))
		if err != nil {
			return err
		}
		err = dir.Sync()
		closeErr = dir.Close()
		return errors.Join(err, closeErr)
	}
	return nil
}
