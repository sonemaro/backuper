package backuper

import (
	"io"

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
