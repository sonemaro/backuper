package backuper

import (
	"io"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/cheggaaa/pb/v3"
	log "github.com/sirupsen/logrus"
)

// IOCopyProgress implements a progress bar on cli for any io.Reader that
// is passed to this func
func IOCopyProgress(r io.Reader, dst io.Writer, limit int64) (err error) {
	bar := pb.Full.Start64(limit)
	barReader := bar.NewProxyReader(r)
	_, err = io.Copy(dst, barReader)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("io.Copy error")
	}
	bar.Finish()
	return
}

// DirFileCount returns number of files(and not dirs)
// in a directory
func DirFileCount(dirPath string) (uint64, error) {
	var ret uint64
	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		atomic.AddUint64(&ret, 1)
		return nil
	}
	err := filepath.Walk(dirPath, walker)
	return ret, err
}
