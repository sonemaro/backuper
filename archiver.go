package backuper

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

// The Archiver provides a set of utility functions in order
// to create archives from files and directories
type Archiver struct{}

// ZipDirectory recursively reads all files in a directory
// and adds them to a zip file
func (*Archiver) ZipDirectory(src, dest string) error {
	return zipDirectory(src, dest)
}

// copied from https://stackoverflow.com/a/63233911
func zipDirectory(src, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		fl, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fl.Close()

		f, err := w.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, fl)
		if err != nil {
			return err
		}

		return nil
	}
	return filepath.Walk(src, walker)
}
