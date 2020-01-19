package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"regexp"
)

// Config json file
type Config struct {
	TrackerURL string `json:"trackerUrl"`
	ConfigURL  string `json:"confUrl"`
	Aria2URL   string `json:"aria2Url"`
	Version    string `json:"version"`
}

func main() {

	dir, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	f := logToFile(dir)
	defer func() {
		err := f.Close()
		if err != nil {
			log.Panic(err)
		}
	}()

	// load BT trackers list url
	config := Config{}

	data, err := ioutil.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		log.Panic(err)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Panic(err)
	}

	// Update aria2 to latest
	url := getLatestAria2DownloadLink(config.Aria2URL)
	if url == "" {
		log.Panicln("can not get latest aria2 release.")
	}
	var isUpdated = false
	s := strings.Split(url, "/")
	fileName := s[len(s)-1]
	version := strings.Split(fileName, "-")[1]
	if isWindows() {
		exe := filepath.Join(dir, "aria2c.exe")
		// Download latest aria2 when aria2.exe not exist or version not match
		if _, er := os.Stat(exe); os.IsNotExist(er) || config.Version != version {

			binaryFile := filepath.Join(dir, fileName)
			if _, err := os.Stat(binaryFile); os.IsNotExist(err) {
				log.Println(binaryFile)
				err = downloadFile(binaryFile, url)
				if err != nil {
					log.Panic(err)
				}
			}

			files, err := unzip(binaryFile, dir)
			if err != nil {
				log.Fatalln(err)
			}

			log.Println("Unzipped:\n" + strings.Join(files, "\n"))

			// Copy files to current dir
			copyFiles(files, dir)

			isUpdated = true
			os.Remove(binaryFile)
			os.RemoveAll(strings.Replace(binaryFile, ".zip", "", 1))
		}
	} else {
		run := exec.Command("which", "aria2c")
		err := run.Run()
		if err != nil {
			if isDarwin() {
				log.Panicf("please install aria2 before run the helper.\n download link(%s)\n", url)
			} else {
				log.Panicln("please install aria2 via 'apt/yum' before run the helper.")
			}
		}
	}

	if isUpdated {
		// Save latest Aria2 version to config.json
		config.Version = version
		data, err := json.MarshalIndent(&config, "", "    ")
		if err != nil {
			log.Panicln(err)
		}

		err = ioutil.WriteFile(filepath.Join(dir, "config.json"), data, 0664)
		if err != nil {
			log.Panicln(err)
		}
	}

	// Get BT Trackers List
	lists := getBTTrackersList(config.TrackerURL)

	// Join list to string
	btTrackers := strings.Join(lists, ",")

	// If aria2.session and aria2.log do not exists, create one
	sessionFile := filepath.Join(dir, "aria2.session")
	if _, er := os.Stat(sessionFile); os.IsNotExist(er) {
		createFile(sessionFile)
	}

	logFile := filepath.Join(dir, "aria2.log")
	if _, er := os.Stat(logFile); os.IsNotExist(er) {
		createFile(logFile)
	}

	confPath := filepath.Join(dir, "aria2.conf")
	var firstLoad bool

	// If aria2.conf does not exist, download it from config repository
	if _, er := os.Stat(confPath); os.IsNotExist(er) {
		if err := downloadFile(confPath, config.ConfigURL); err != nil {
			log.Panic(err)
		}
		firstLoad = true
	}

	aria2Config := loadAria2Config(confPath)

	if !strings.Contains(aria2Config, btTrackers) {

		p := regexp.MustCompile(`(bt-tracker=.*)`)
		data := p.ReplaceAllString(aria2Config, "bt-tracker="+btTrackers)
		// If first time download aria2.conf, setup below dirs
		if firstLoad {
			p = regexp.MustCompile(`(dir=.*)`)
			if isWindows() {
				data = p.ReplaceAllString(data, "dir=D:\\Downloads")
			} else {
				data = p.ReplaceAllString(data, "dir=~/Downloads")
			}
			p = regexp.MustCompile(`(log=.*)`)
			data = p.ReplaceAllString(data, "log="+logFile)
			p = regexp.MustCompile(`(input-file=.*)`)
			data = p.ReplaceAllString(data, "input-file="+sessionFile)
			p = regexp.MustCompile(`(save-session=.*)`)
			data = p.ReplaceAllString(data, "save-session="+sessionFile)
		}
		err := ioutil.WriteFile(confPath, []byte(data), 0644)
		if err != nil {
			log.Panic(err)
		}
	}

	// Start aria2c.exe
	var run *exec.Cmd
	if isWindows() {
		run = exec.Command("aria2c.exe", "--conf-path=aria2.conf")
	} else {
		run = exec.Command("aria2c", "--conf-path=aria2.conf")
	}

	out, err := run.Output()
	if err != nil {
		panic(err)
	}
	log.Println(out)
	log.Println("start success!")
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func isDarwin() bool {
	return runtime.GOOS == "darwin"
}

func loadAria2Config(path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Panic(err)
	}
	return string(data)
}

func createFile(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Panic(err)
	}
	if err := f.Close(); err != nil {
		log.Panic(err)
	}
}

func getBTTrackersList(url string) []string {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}

	defer func() {
		e := resp.Body.Close()
		if e != nil {
			log.Fatalln(e)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	trackers := make([]string, 0)
	for _, v := range strings.Split(string(body), "\n") {
		if v != "" {
			trackers = append(trackers, v)
		}
	}
	return trackers
}

func downloadFile(filepath string, url string) (err error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		e := resp.Body.Close()
		if e != nil {
			err = e
		}
	}()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		e := out.Close()
		if e != nil {
			err = e
		}
	}()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Panic(err)
	}
	return
}

func getLatestAria2DownloadLink(url string) string {

	c, err := http.Get(url)
	if err != nil {
		log.Panicln(err)
	}
	defer func() {
		e := c.Body.Close()
		if e != nil {
			log.Panicln(e)
		}
	}()

	body, err := ioutil.ReadAll(c.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var p *regexp.Regexp
	if isWindows() {
		p = regexp.MustCompile(`https:.*\.zip`)
		for _, v := range strings.Split(string(body), ",") {
			if strings.Contains(v, "win-64bit-build1.zip") && strings.Contains(v, "browser_download_url") {
				url := p.FindString(v)
				return url
			}
		}
	} else {
		p = regexp.MustCompile(`https:.*\.dmg`)
		for _, v := range strings.Split(string(body), ",") {
			if strings.Contains(v, ".dmg") && strings.Contains(v, "browser_download_url") {
				url := p.FindString(v)
				return url
			}
		}

	}
	return ""
}

func copyFile(src, dst string) (nBytes int64, err error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer func() {
		if e := source.Close(); e != nil {
			nBytes, err = 0, e
		}
	}()

	destination, err := os.Create(filepath.Join(dst, sourceFileStat.Name()))
	if err != nil {
		return 0, err
	}

	defer func() {
		if e := destination.Close(); e != nil {
			nBytes, err = 0, e
		}
	}()
	nBytes, err = io.Copy(destination, source)
	return
}

func copyFiles(files []string, dst string) {

	for _, src := range files {
		sourceFileStat, err := os.Stat(src)
		if err != nil {
			log.Panicln(err)
		}

		if sourceFileStat.Mode().IsDir() {
			continue
		}
		bytes, err := copyFile(src, dst)
		if err != nil {
			log.Panicln(err)
		}
		log.Printf("copied: %d bytes.", bytes)
	}
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func unzip(src string, dest string) (filenames []string, err error) {

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer func() {
		if e := r.Close(); e != nil {
			err = e
		}
	}()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		if e := outFile.Close(); e != nil {
			log.Fatalln(e)
		}
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return
}

func logToFile(dir string) *os.File {

	//create log file name
	var filename = "info-%s.log"
	format := strings.Replace(time.Now().Format(time.RFC3339), ":", "-", -1)
	filename = fmt.Sprintf(filename, format)
	file := filepath.Join(dir, filename)

	//create your file with desired read/write permissions
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalln(err)
	}

	//set output of logs to f
	log.SetOutput(f)
	return f
}
