package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"

	klog "github.com/go-kit/kit/log"
	bimg "gopkg.in/h2non/bimg.v1"
)

const gCloud = "http://storage-download.googleapis.com/img.meepcloud.com"

type requestInfo struct {
	uri    string
	width  int
	height int
}

const version = "v1.5.0"

var allowedImageTypes = map[string]string{
	"image/gif":  "gif",
	"image/png":  "png",
	"image/jpg":  "jpg",
	"image/jpeg": "jpeg",
	"image/webp": "webp",
	"image/tiff": "tiff",
}

var httpClient *http.Client
var logger klog.Logger

func main() {

	log.Println("resizer start. version: ", version)

	logger = klog.NewLogfmtLogger(os.Stdout)
	logger = klog.With(logger, "caller", klog.DefaultCaller)

	httpClient = &http.Client{Timeout: time.Duration(30 * time.Second)}

	http.HandleFunc("/", resizer)
	http.HandleFunc("/healthz", healthCheck)
	http.HandleFunc("/version", versionCheck)

	log.Fatal(http.ListenAndServe(":4321", nil))
}

func resizer(w http.ResponseWriter, r *http.Request) {
	var buf []byte
	var err error

	request := requestInfo{}

	// get params
	u, _ := url.Parse(r.RequestURI)
	request.uri = gCloud + u.Path
	params := u.Query()
	if params.Get("w") != "" {
		request.width, err = strconv.Atoi(params.Get("w"))
	} else if params.Get("h") != "" {
		request.height, err = strconv.Atoi(params.Get("h"))
	}
	if err != nil {
		logger.Log("uri", r.RequestURI, "err", err.Error())

		w.WriteHeader(400)
		w.Write([]byte("size w or h error"))
		return
	}

	// get header
	resp, err := httpClient.Head(request.uri)
	if err != nil {
		logger.Log("uri", r.RequestURI, "err", err.Error())

		w.WriteHeader(400)
		w.Write([]byte("get head fail"))
		return
	}
	if resp.StatusCode != 200 {
		logger.Log("uri", r.RequestURI, "status", resp.StatusCode)

		w.WriteHeader(400)
		w.Write([]byte("get head fail"))
		return
	}

	imgType, allow := allowedImageTypes[resp.Header.Get("Content-Type")]
	if allow && (request.width > 0 || request.height > 0) {

		// TEST
		// start := time.Now()

		if imgType == "gif" {
			buf = imageMagickResize(&request)
		} else {
			buf = bimgResize(&request)
			if len(buf) == 0 {
				w.WriteHeader(400)
				w.Write([]byte("get image fail"))
				return
			}
		}

		// TEST
		// elapsed := time.Since(start)
		// log.Printf("took %s", elapsed)
	}

	if len(buf) == 0 {
		buf, err = getImage(&request)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("get image fail"))
			return
		}
	}

	w.Header().Add("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Add("Cache-Control", "public, max-age=31536000")
	w.Header().Add("Expires", resp.Header.Get("Expires"))
	w.Write(buf)
}

func imageMagickResize(req *requestInfo) []byte {
	size := ""
	if req.width > 0 {
		size = fmt.Sprintf("%dx>", req.width)
	} else if req.height > 0 {
		size = fmt.Sprintf("x%d>", req.height)
	}

	imageMagickCmd, err := exec.LookPath("convert")
	if err != nil {
		logger.Log("uri", req.uri, "err", err.Error())
		return []byte{}
	}

	cmd := exec.Command(imageMagickCmd, req.uri, "-coalesce", "-scale", size, "-")

	cmd.Stderr = os.Stderr
	stdout, err := cmd.Output()
	if err != nil {
		logger.Log("uri", req.uri, "err", err.Error())
		return []byte{}
	}

	return stdout
}

func bimgResize(req *requestInfo) []byte {
	start := time.Now()
	defer func() {
		fmt.Println(time.Since(start))
	}()

	body, err := getImage(req)
	if err != nil {
		return []byte{}
	}

	// 檢查圖片原尺寸 不做放大處理
	width, height, err := bimg.GetImageWH(body)
	if err != nil {
		logger.Log("uri", req.uri, "err", err.Error())
		return body
	}
	if (req.width > 0 && req.width > width) || (req.height > 0 && req.height > height) {
		return body
	}

	options := bimg.Options{
		Width:   req.width,
		Height:  req.height,
		Quality: int(85),
	}

	buf, err := bimg.Resize(body, options)
	if err != nil {
		logger.Log("uri", req.uri, "err", err.Error())
		return body
	}

	return buf
}

func getImage(req *requestInfo) ([]byte, error) {

	resp, err := httpClient.Get(req.uri)
	if err != nil {
		logger.Log("uri", req.uri, "err", err.Error())
		return []byte{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Log("uri", req.uri, "err", err.Error())
		return []byte{}, err
	}
	resp.Body.Close()

	return body, nil
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Add("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Add("Pragma", "no-cache")
	h.Add("Expires", "0")
	fmt.Fprintf(w, "OK")
}

func versionCheck(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Add("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Add("Pragma", "no-cache")
	h.Add("Expires", "0")
	fmt.Fprintf(w, version)
}
