package loader

import (
	"archive/zip"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
)

func download(url, dir string) error {
	switch url {

	case "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevResourceDownloadTo&Category=rail", "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevResourceDownloadTo&Category=bus":
		return njtdl(url, dir)

	default:
		return defaultDL(url, dir)
	}
}

func njtdl(url, dir string) error {
	return errors.New("arghhh")
}

func defaultDL(url, dir string) error {

	// Download and save file, opening it for writing (web response) and
	// reading (unzipper)
	fh, err := ioutil.TempFile(dir, "")
	if err != nil {
		return err
	}
	defer fh.Close()
	defer os.Remove(fh.Name())

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	n, err := io.Copy(fh, resp.Body)
	if err != nil {
		return err
	}

	// Flush and reset file for reading
	err = fh.Sync()
	if err != nil {
		return err
	}
	_, err = fh.Seek(0, 0)
	if err != nil {
		return err
	}

	// Create a zip reader
	z, err := zip.NewReader(fh, n)
	if err != nil {
		return err
	}

	// Go through each file in the zip, using a closure so we can
	// defer closing of each file
	for _, f := range z.File {
		func() {
			// Create a file to write to
			zwfh, err := os.Create(path.Join(dir, f.Name))
			if err != nil {
				return
			}
			defer zwfh.Close()

			// Read the file in the zip
			zrfh, err := f.Open()
			if err != nil {
				return
			}
			defer zrfh.Close()

			// Write from zip file to actual file
			_, err = io.Copy(zwfh, zrfh)
			if err != nil {
				return
			}
		}()
		if err != nil {
			return err
		}
	}

	return nil
}
