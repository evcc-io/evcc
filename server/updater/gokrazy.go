//go:build gokrazy

package updater

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/gokrazy/updater"
)

// update constants
const (
	mb         = 1024 * 1024
	rootOffset = 8192*512 + 100*mb
	rootSize   = 500 * mb
)

var (
	Host     = "localhost"
	Port     = 8080
	Password = "SECRET"
)

// unzipReader transparently unpacks zip files
func unzipReader(file io.ReadCloser) (io.ReadCloser, error) {
	if unzipped, err := gzip.NewReader(file); err == nil {
		return unzipped, nil
	}

	return file, nil
}

type countingWriter struct {
	count int
	C     chan int
}

func (cw *countingWriter) Write(p []byte) (n int, err error) {
	cw.count += len(p)
	cw.C <- cw.count
	return len(p), nil
}

var mutex int32

// Update request handler
func (u *watch) execute(assetID int64, size int) error {
	if !atomic.CompareAndSwapInt32(&mutex, 0, 1) {
		return errors.New("upgrade already running")
	}

	rootFS, err := u.repo.StreamAsset(assetID)
	if err != nil {
		return err
	}

	if rootFS, err = unzipReader(rootFS); err != nil {
		rootFS.Close()
		return err
	}

	uri := fmt.Sprintf("http://gokrazy:%s@%s:%d/", Password, Host, Port)
	target, err := updater.NewTarget(context.TODO(), uri, http.DefaultClient)
	if err != nil {
		return err
	}

	// stream async to device
	go u.executeAsync(target, rootFS, size)

	return nil
}

func (u *watch) executeAsync(target *updater.Target, rootFS io.ReadCloser, size int) error {
	defer func() {
		atomic.StoreInt32(&mutex, 0)
		rootFS.Close()
	}()

	cw := &countingWriter{C: make(chan int)}

	go func() {
		for v := range cw.C {
			u.Send("uploadProgress", 100*v/size)
		}
		u.Send("uploadProgress", 100)
	}()

	u.Send("uploadMessage", "uploading")
	if err := target.StreamTo(context.TODO(), "root", io.TeeReader(rootFS, cw)); err != nil {
		return fmt.Errorf("updating root file system: %w", err)
	}
	close(cw.C) // upload finished

	u.Send("uploadMessage", "switching to non-active partition")
	if err := target.Switch(context.TODO()); err != nil {
		return fmt.Errorf("switching to non-active partition: %w", err)
	}

	u.Send("uploadMessage", "rebooting")
	if err := target.Reboot(context.TODO()); err != nil {
		return fmt.Errorf("reboot: %w", err)
	}

	return nil
}
