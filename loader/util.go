package loader

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
)

func getcsv(dir, name string) (*csv.Reader, io.Closer) {

	f, err := os.Open(path.Join(dir, name))
	if err != nil {
		panic(err)
	}

	r := csv.NewReader(f)
	r.LazyQuotes = true

	return r, f
}

func writecsvtmp(dir string) (*csv.Writer, *os.File) {
	outFH, err := ioutil.TempFile(dir, "")
	if err != nil {
		log.Fatal(err)
	}

	w := csv.NewWriter(outFH)

	return w, outFH
}

// find index of col in header or panic
func find(header []string, col string) int {
	idx := maybeFind(header, col)
	if idx >= 0 {
		return idx
	}

	panic(fmt.Sprintf("can't find header col %v", col))
}

// maybeFind index of col in header, return -1 if not found
func maybeFind(header []string, col string) int {
	for i := 0; i < len(header); i++ {
		if header[i] == col {
			return i
		}
	}

	return -1
}

type rewrite struct {
	inFH *os.File
	r    *csv.Reader

	outFH *os.File
	w     *csv.Writer

	header   []string
	filepath string
}

func (rw *rewrite) finish() (err error) {
	rw.w.Flush()

	err = rw.outFH.Close()
	if err != nil {
		return err
	}

	return os.Rename(rw.outFH.Name(), rw.filepath)
}

func (rw *rewrite) clean() (err error) {
	err = rw.outFH.Close()
	if err != nil {
		return err
	}

	return os.Remove(rw.outFH.Name())
}

func newRewrite(dir, filename string) (rw *rewrite, err error) {
	rw = &rewrite{}

	rw.filepath = path.Join(dir, "routes.txt")
	rw.inFH, err = os.Open(rw.filepath)
	if err != nil {
		return
	}
	rw.r = csv.NewReader(rw.inFH)
	rw.r.LazyQuotes = true

	// Create an outgoing csv file for transformed data
	rw.w, rw.outFH = writecsvtmp(dir)

	// Read the existing header
	rw.header, err = rw.r.Read()
	if err != nil {
		return
	}

	return
}
