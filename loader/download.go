package loader

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/brnstz/bus/internal/conf"
)

func download(dlURL, dir string) error {
	switch dlURL {

	case "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevResourceDownloadTo&Category=rail", "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevResourceDownloadTo&Category=bus":
		return njtdl(dlURL, dir)

	default:
		return defaultDL(dlURL, dir)
	}
}

func njtdl(dlURL, dir string) error {
	var err error
	var sessionID string
	login := "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevLoginSubmitTo"

	/*

		client := http.Client{}
		client.Jar, err = cookiejar.New(nil)
		if err != nil {
			return err
		}

		landing := "https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevLoginTo"

		// Get initial page headers
		req, err := http.NewRequest("GET", landing, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		err = resp.Body.Close()
		if err != nil {
			return err
		}

		params := url.Values{}
		params.Set("userName", conf.Loader.NJTransitFeedUsername)
		params.Set("password", conf.Loader.NJTransitFeedPassword)

		log.Println("COOKIES", client.Jar)

		// Get the page to submit login to
		req, err = http.NewRequest("POST", login, bytes.NewBufferString(params.Encode()))
		if err != nil {
			return err
		}
		req.Header.Set("Referer", landing)

		// Run login
		resp, err = client.Do(req)
		if err != nil {
			return err
		}

		log.Println("COOKIES", client.Jar)

		err = resp.Body.Close()
		if err != nil {
			return err
		}

	*/

	/*
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/52.0.2743.116 Safari/537.36")
		req.Header.Set("Referer", login)
	*/

	params := url.Values{}
	params.Set("userName", conf.Loader.NJTransitFeedUsername)
	params.Set("password", conf.Loader.NJTransitFeedPassword)

	// Get the page to submit login to
	req, err := http.NewRequest("POST", login, bytes.NewBufferString(params.Encode()))
	if err != nil {
		return err
	}
	// Run login
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}
	for _, v := range resp.Cookies() {
		log.Println("HELLO", v.Name, v.Value)
		if v.Name == "JSESSIONID" {
			sessionID = v.Value
		}
	}

	// Get the actual download page
	req, err = http.NewRequest("GET", dlURL, nil)
	if err != nil {
		return err
	}
	log.Println("what is the sessionID?", sessionID)
	req.AddCookie(&http.Cookie{Name: "JSESSIONID", Value: sessionID})

	// Run login
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//log.Println("COOKIES", client.Jar)
	b, err := ioutil.ReadAll(resp.Body)

	log.Printf("resp says: %v %s", err, b)

	return nil

	//curl 'https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevLoginSubmitTo' -H 'Cookie: JSESSIONID=1EE203692B6CE6128B256EB8786B6078; _ga=GA1.2.1127045342.1470930152; __utmz=190986997.1470945795.3.2.utmccn=(referral)|utmcsr=google.com|utmcct=/|utmcmd=referral; __utma=190986997.1127045342.1470930152.1470945795.1470947756.4; __utmc=190986997; __utmb=190986997' -H 'Origin: https://www.njtransit.com' -H 'Accept-Encoding: gzip, deflate, br' -H 'Accept-Language: en-US,en;q=0.8' -H 'Upgrade-Insecure-Requests: 1' -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/52.0.2743.116 Safari/537.36' -H 'Content-Type: application/x-www-form-urlencoded' -H 'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8' -H 'Cache-Control: max-age=0' -H 'Referer: https://www.njtransit.com/mt/mt_servlet.srv?hdnPageAction=MTDevLoginTo' -H 'Connection: keep-alive' --data 'userName=brnstz&password=d4ndr0' --compressed

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
