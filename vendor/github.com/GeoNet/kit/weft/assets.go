package weft

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type asset struct {
	path       string
	hashedPath string
	mime       string
	b          []byte
}

// assets and assetPath is populated during init and then is only used for reading.
var assets = make(map[string]*asset)
var assetPath = make(map[string]string)
var assetError error

func init() {
	assetError = initAssets("assets/assets", "assets")
}

// AssetHandler serves assets from the local directory `assets/assets`.  Assets are loaded from this
// directory on start up and served from memory.  Any errors during start up will be served by AssetHandlers.
// Assets are served at the path `/assets/...` and can be also be served with a hashed path which finger prints the asset
// for uniqueness for caching e.g.,
//
//    /assets/bootstrap/hello.css
//    /assets/bootstrap/1fdd2266-hello.css
//
// The finger printed path can be looked up with AssetPath.
func AssetHandler(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := CheckQuery(r, []string{"GET"}, []string{}, []string{"v"})
	if err != nil {
		return err
	}

	if assetError != nil {
		return assetError
	}

	a := assets[r.URL.Path]
	if a == nil {
		return StatusError{Code: http.StatusNotFound}
	}

	b.Write(a.b)

	h.Set("Surrogate-Control", "max-age=86400")
	h.Set("Cache-Control", "max-age=86400")
	h.Set("Content-Type", a.mime)

	return nil
}

// AssetPath returns the finger printed path for path e.g., `/assets/bootstrap/hello.css`
// returns `/assets/bootstrap/1fdd2266-hello.css`.
func AssetPath(path string) string {
	return assetPath[path]
}

// loadAsset loads file and finger prints it with a sha256 hash.  prefix is stripped
// from path members in the returned asset.
func loadAsset(file, prefix string) (*asset, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// calculate a hash for the file and prefix the asset name with a short hash.
	h := sha256.New()

	_, err = io.Copy(h, f)
	if err != nil {
		return nil, err
	}

	a := asset{
		path: strings.TrimPrefix(file, prefix),
	}

	var suffix string
	l := strings.LastIndex(a.path, ".")
	if l > -1 && l < len(a.path) {
		suffix = strings.ToLower(a.path[l+1:])
	}

	// these types should appear in weft.compressibleMimes as appropriate
	switch suffix {
	case "js":
		a.mime = "text/javascript"
	case "css", "map":
		a.mime = "text/css"
	case "jpeg", "jpg":
		a.mime = "image/jpeg"
	case "png":
		a.mime = "image/png"
	case "gif":
		a.mime = "image/gif"
	case "ico":
		a.mime = "image/x-icon"
	case "ttf", "woff", "woff2":
		a.mime = "application/octet-stream"
	case "json":
		a.mime = "application/json"
	}

	p := strings.Split(a.path, "/")
	p[len(p)-1] = fmt.Sprintf("%x-%s", h.Sum(nil)[0:4], p[len(p)-1])

	a.hashedPath = strings.Join(p, "/")

	_, err = f.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	a.b, err = ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return &a, nil
}

// initAssets loads all assets below dir into global maps.
func initAssets(dir, prefix string) error {
	var fileList []string

	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		fileList = append(fileList, path)
		return nil
	})
	if err != nil {
		return err
	}

	for _, v := range fileList {
		fi, err := os.Stat(v)
		if err != nil {
			return err
		}

		switch mode := fi.Mode(); {
		case mode.IsRegular():
			a, err := loadAsset(v, prefix)
			if err != nil {
				return err
			}

			assets[a.path] = a
			assets[a.hashedPath] = a
			assetPath[a.path] = a.hashedPath
		}
	}

	return nil
}
