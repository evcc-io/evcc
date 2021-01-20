// +build gokrazy

package updater

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/gokrazy/updater"
)

// update constants
const (
	MB         = 1024 * 1024
	RootOffset = 8192*512 + 100*MB
	RootSize   = 500 * MB
	RootFS     = "evcc_%s.rootfs.gz"
)

var (
	Password = "SECRET"
	Port     = 8080
)

// unzipReader transparently unpacks zip files
func unzipReader(file io.ReadCloser) (io.ReadCloser, error) {
	if unzipped, err := gzip.NewReader(file); err == nil {
		return unzipped, err
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
	defer atomic.StoreInt32(&mutex, 0)

	rootFS, err := u.repo.StreamAsset(assetID)
	if err != nil {
		return err
	}

	if rootFS, err = unzipReader(rootFS); err != nil {
		rootFS.Close()
		return err
	}
	defer rootFS.Close()

	cw := &countingWriter{C: make(chan int)}

	go func() {
		for v := range cw.C {
			u.Send("uploadProgress", 100*v/size)
		}
		u.Send("uploadProgress", 100)
	}()

	uri := fmt.Sprintf("http://gokrazy:%s@localhost:%d/", Password, Port)
	target, err := updater.NewTarget(uri, http.DefaultClient)
	if err != nil {
		return err
	}

	u.Send("uploadMessage", "uploading")
	if err := target.StreamTo("root", io.TeeReader(rootFS, cw)); err != nil {
		return fmt.Errorf("updating root file system: %w", err)
	}
	close(cw.C) // upload finished

	u.Send("uploadMessage", "switching to non-active partition")
	if err := target.Switch(); err != nil {
		return fmt.Errorf("switching to non-active partition: %w", err)
	}

	u.Send("uploadMessage", "rebooting")
	if err := target.Reboot(); err != nil {
		return fmt.Errorf("reboot: %w", err)
	}

	return nil
}
