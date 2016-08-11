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
