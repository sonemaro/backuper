package backuper

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

// The Archiver provides a set of utility functions in order
// to create archives from files and directories
type Archiver struct {
	// BeforeCopyCallback will be called before each io.Copy
	// in walker. The purpose is to implement progress bar or
	// any other calculation outside of this package
	BeforeCopyCallback func(path string, info os.FileInfo)

	// AfterCopyCallback will be called after each SUCCESSFUL io.Copy
	// in walker. The purpose is to implement progress bar or
	// any other calculation outside of this package
	AfterCopyCallback func(path string, info os.FileInfo)
}

// NewArchiver returns a fresh instance of Archiver
// with a simple IOCopyProxy. Default IOCopyProxy
// runs a normal io.Copy without anything special.
func NewArchiver() *Archiver {
	return &Archiver{
		BeforeCopyCallback: func(path string, info os.FileInfo) {},
		AfterCopyCallback:  func(path string, info os.FileInfo) {},
	}
}

// ZipDirectory recursively reads all files in a directory
// and adds them to a zip file
// copied from https://stackoverflow.com/a/63233911
func (a *Archiver) ZipDirectory(src, dest string) error {
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

		a.BeforeCopyCallback(path, info)
		_, err = io.Copy(f, fl)

		if err != nil {
			return err
		}
		a.AfterCopyCallback(path, info)

		return nil
	}
	return filepath.Walk(src, walker)
}

// ZipFiles compresses one or many files into a single zip archive file.
// Param 1: filename is the output zip file's name.
// Param 2: files is a list of files to add to the zip.
// returns os.FileInfo of the archive and an error
func ZipFiles(filename string, files []string) (os.FileInfo, error) {
	newZipFile, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {
		if err = addFileToZip(zipWriter, file); err != nil {
			return nil, err
		}
	}
	return newZipFile.Stat()
}

func addFileToZip(zipWriter *zip.Writer, filename string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = filename
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}
