package loader

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/brnstz/bus/internal/conf"
)

func download(dlURL, dir string) error {
	switch dlURL {

	case "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevResourceDownloadTo&Category=rail", "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevResourceDownloadTo&Category=bus":
		return njtDL(dlURL, dir)

	case "https://github.com/septadev/GTFS/releases/latest":
		return phillyDL(dlURL, "rail", dir)

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

	// Run login and close body (we don't need it)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}

	// Get the cookie value
	for _, v := range resp.Cookies() {
		if v.Name == "JSESSIONID" {
			sessionID = v.Value
		}
	}
	if len(sessionID) < 1 {
		return errors.New("no session ID in NJT response")
	}

	// Try sleeping a bit between login and request, this seems to
	// fail randomly sometimes.
	time.Sleep(5 * time.Second)

	// Get the actual download page and add our session cookie
	req, err = http.NewRequest("GET", dlURL, nil)
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{Name: "JSESSIONID", Value: sessionID})

	// Run download, get zipfile response and send to unzip
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	buff := bytes.NewReader(b)

	return unzipit(dir, buff, int64(len(b)))
}

func phillyDL(dlURL, subZip string, dir string) error {
	var latestURL string

	// Get the GH index page
	resp, err := http.Get(dlURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	z := html.NewTokenizer(resp.Body)

	// Find the first anchor token with gtfs_public.zip as the filename
	for {
		tt := z.Next()

		// If we get this, it's an error in the text or we reached
		// io.EOF without finding our link. In both cases, a fatal
		// error for us.
		if tt == html.ErrorToken {
			return z.Err()
		}

		if tt == html.StartTagToken {
			name, hasAttr := z.TagName()
			if string(name) != "a" {
				continue
			}
			if !hasAttr {
				continue
			}

			for {
				key, val, more := z.TagAttr()
				if string(key) == "href" && strings.HasSuffix(string(val), "gtfs_public.zip") {
					latestURL = "https://github.com" + string(val)
				}

				if !more {
					break
				}
			}
		}

		if len(latestURL) > 0 {
			break
		}
	}

	// Create a special temp subdir so we can extract out the
	// special philly dir we want
	subDir, err := ioutil.TempDir(conf.Loader.TmpDir, "")
	if err != nil {
		log.Println(err)
		return err
	}
	defer os.RemoveAll(subDir)

	err = defaultDL(latestURL, subDir)
	if err != nil {
		log.Println(err)
		return err
	}

	// FIXME we need to unzip the subzip file first

	// Move all files from subDir/subZip into dir
	fullSubDir := path.Join(subDir, subZip)
	files, err := ioutil.ReadDir(fullSubDir)
	if err != nil {
		log.Println(err)
		return err
	}
	for _, f := range files {
		if !f.IsDir() {
			err = os.Rename(
				path.Join(fullSubDir, f.Name()),
				path.Join(dir, f.Name()),
			)
			if err != nil {
				log.Println(err)
				return err
			}
		}
	}

	return nil
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
