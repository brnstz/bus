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

// find index of col in header
func find(header []string, col string) int {
	for i := 0; i < len(header); i++ {
		if header[i] == col {
			return i
		}
	}

	panic(fmt.Sprintf("can't find header col %v", col))
	return -1
}
