package backuper

import (
	"archive/zip"
	"fmt"
	"testing"

	"github.com/spf13/afero"
)

func TestZipDirectory(t *testing.T) {
	af := afero.NewOsFs()
	n, err := afero.TempDir(af, "", "tttt")
	if err != nil {
		t.Error(err)
	}

	numFiles := 10
	char := "a"
	for i := 0; i < numFiles; i++ {
		name := fmt.Sprintf("tst_%d", i)
		f, err := afero.TempFile(af, n, name)
		if err != nil {
			t.Error(err)
		}
		_, err = f.WriteString(char)
		if err != nil {
			t.Error(err)
		}
		f.Close()
	}

	p := "/tmp/test_dist_zip.zip"
	arc := Archiver{}
	err = arc.ZipDirectory(n, p)
	if err != nil {
		t.Error(err)
	}

	r, err := zip.OpenReader(p)
	if err != nil {
		t.Error(err)
	}
	defer r.Close()

	// check files of our final archive
	for _, f := range r.File {
		if !f.FileInfo().IsDir() && f.FileInfo().Size() != int64(len(char)) {
			t.Error("invalid file size")
		}
	}

	af.RemoveAll(n)
}
