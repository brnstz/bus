package loader

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/brnstz/bus/internal/conf"
)

func download(dlURL, dir string) error {
	switch dlURL {

	case "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevResourceDownloadTo&Category=rail", "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevResourceDownloadTo&Category=bus":
		return njtDL(dlURL, dir)

	default:
		return defaultDL(dlURL, dir)
	}
}

func unzipit(dir string, r io.ReaderAt, n int64) error {
	// Create a zip reader
	z, err := zip.NewReader(r, n)
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

func njtDL(dlURL, dir string) error {
	var err error
	var sessionID string
	login := "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevLoginSubmitTo"

	params := url.Values{}
	params.Set("userName", conf.Loader.NJTransitFeedUsername)
	params.Set("password", conf.Loader.NJTransitFeedPassword)

	// Get the page to submit login to
	req, err := http.NewRequest(
		"POST", login, bytes.NewBufferString(params.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Run login
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	for _, v := range resp.Cookies() {
		if v.Name == "JSESSIONID" {
			sessionID = v.Value
		}
	}

	// Get the actual download page
	req, err = http.NewRequest("GET", dlURL, nil)
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{Name: "JSESSIONID", Value: sessionID})

	// Run login
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	buff := bytes.NewReader(b)

	return unzipit(dir, buff, int64(len(b)))
}

func defaultDL(dlURL, dir string) error {

	// Download and save file, opening it for writing (web response) and
	// reading (unzipper)
	fh, err := ioutil.TempFile(dir, "")
	if err != nil {
		return err
	}
	defer fh.Close()
	defer os.Remove(fh.Name())

	resp, err := http.Get(dlURL)
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

	return unzipit(dir, fh, n)
}
