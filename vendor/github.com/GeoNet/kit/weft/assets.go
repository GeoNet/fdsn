package weft

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type asset struct {
	path       string
	hashedPath string
	mime       string
	fileType   string
	b          []byte
	sri        string
}

// assets is populated during init and then is only used for reading.
var assets = make(map[string]*asset)

// assetHashes maps asset filename to the corresponding hash-prefixed asset pathname.
var assetHashes = make(map[string]string)
var assetError error

func init() {
	assetError = initAssets("assets/assets", "assets")
}

// As part of Subresource Integrity we need to calculate the hash of the asset, we do this when the asset is loaded into memory
// This should only be used for files that are stored alongside the server, as remote files could be tampered with and we'd still
// just calculate the hash.
// Externally hosted files should have a precalculated SRI
func calcSRIhash(b []byte) (string, error) {
	var buf bytes.Buffer

	dgst := sha512.Sum384(b)

	enc := base64.NewEncoder(base64.StdEncoding, &buf)
	_, err := enc.Write(dgst[:])
	if err != nil {
		return "", fmt.Errorf("failed to encode SRI hash: %v", err)
	}

	return "sha384-" + buf.String(), nil
}

// getCspNonce creates a random nonce string of specified length.
func getCspNonce(len int) (string, error) {
	b := make([]byte, len)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", fmt.Errorf("failed to create nonce: %v", err)
	}
	var buf bytes.Buffer
	enc := base64.NewEncoder(base64.StdEncoding, &buf)
	defer enc.Close()
	_, err := enc.Write(b)
	if err != nil {
		return "", fmt.Errorf("failed to create nonce: %v", err)
	}
	return buf.String(), nil
}

// createSubResourceTag returns a script tag as a string, based on the given
// asset, nonce, and script loading attribute (ie: "defer" or "async")
func createSubResourceTag(a *asset, nonce, attr string) (string, error) {
	switch a.fileType {
	case "js":
		if nonce != "" {
			return fmt.Sprintf(`<script src="%s" type="text/javascript" integrity="%s" nonce="%s" %s></script>`, a.hashedPath, a.sri, nonce, attr), nil
		} else {
			return fmt.Sprintf(`<script src="%s" type="text/javascript" integrity="%s" %s></script>`, a.hashedPath, a.sri, attr), nil
		}
	case "mjs":
		if nonce != "" {
			return fmt.Sprintf(`<script src="%s" type="module" integrity="%s" nonce="%s" %s></script>`, a.hashedPath, a.sri, nonce, attr), nil
		} else {
			return fmt.Sprintf(`<script src="%s" type="module" integrity="%s" %s></script>`, a.hashedPath, a.sri, attr), nil
		}
	case "css":
		return fmt.Sprintf(`<link rel="stylesheet" href="%s" integrity="%s" %s>`, a.hashedPath, a.sri, attr), nil
	case "map":
		return fmt.Sprintf(`<link rel="stylesheet" href="%s" integrity="%s" %s>`, a.path, a.sri, attr), nil
	default:
		return "", fmt.Errorf("cannot create an embedded resource tag for mime: '%v'", a.mime)
	}
}

// createSubResourcePreloadTag returns a <link> module preload tag for a .mjs file.
func createSubResourcePreloadTag(a *asset, nonce string) (string, error) {
	if a.fileType != "mjs" {
		return "", errors.New("can only create module preload tag for module scripts")
	}
	if nonce != "" {
		return fmt.Sprintf(`<link rel="modulepreload" href="%s" integrity="%s" nonce="%s"/>`, a.hashedPath, a.sri, nonce), nil
	} else {
		return fmt.Sprintf(`<link rel="modulepreload" href="%s" integrity="%s"/>`, a.hashedPath, a.sri), nil
	}
}

// CreateSubResourceTag generates a tag for a resource with the hashed path and SRI hash.
// Returns a template.HTML so it won't throw warnings with golangci-lint.
// args can be 1~3 strings: 1. the asset path, 2. nonce for script attribute,
// 3. script loading attribute ("defer" or "async").
func CreateSubResourceTag(args ...string) (template.HTML, error) {
	var nonce string
	if len(args) > 1 {
		nonce = args[1]
	}
	var attr string
	if len(args) > 2 {
		if args[2] == "defer" || args[2] == "async" {
			attr = args[2]
		}
	}
	hashedPath, ok := assetHashes[args[0]]
	if !ok {
		return template.HTML(""), fmt.Errorf("hashed pathname for asset not found for '%s", args[0])
	}
	a, ok := assets[hashedPath]
	if !ok {
		return template.HTML(""), fmt.Errorf("asset does not exist at path '%v'", hashedPath)
	}

	s, err := createSubResourceTag(a, nonce, attr)

	return template.HTML(s), err //nolint:gosec //We're writing these ourselves, any changes will be reviewd, acceptable risk. (Could add URLencoding if there's any concern)
}

// CreateSubResourcePreload generates a tag that preloads a JavaScript module file. This is helpful to
// allow the file to be fetched in parallel with the module file that imports it, and also allows us
// to set the SRI attribute of imported modules.
func CreateSubResourcePreload(args ...string) (template.HTML, error) {
	var nonce string
	if len(args) > 1 {
		nonce = args[1]
	}
	hashedPath, ok := assetHashes[args[0]]
	if !ok {
		return template.HTML(""), fmt.Errorf("hashed pathname for asset not found for '%s", args[0])
	}
	a, ok := assets[hashedPath]
	if !ok {
		return template.HTML(""), fmt.Errorf("asset does not exist at path '%v'", hashedPath)
	}

	s, err := createSubResourcePreloadTag(a, nonce)

	return template.HTML(s), err //nolint:gosec
}

// CreateImportMap generates an import map script tag which maps JS module asset filenames to their
// respectful hash-prefixed path name. eg:
//
//	<script type="importmap" nonce="abcdefghijklmnop">
//	{
//		"imports":{
//			"geonet-map.mjs":"/assets/js/77da7c4e-geonet-map.mjs"
//		}
//	}
//	</script>
func CreateImportMap(nonce string) template.HTML {

	importMapping := make(map[string]string, 0)
	for k, v := range assetHashes {
		if !strings.HasSuffix(k, ".mjs") {
			continue
		}
		filename := path.Base(k)
		importMapping[filename] = v
	}
	if len(importMapping) == 0 {
		return template.HTML("")
	}
	importMap := createImportMapTag(importMapping, nonce)

	return template.HTML(importMap) //nolint:gosec
}

// createImportMapTag returns the <script> tag of type "importmap" to faciliate browser with
// module resolution. Formatted to make readable in resulting source file.
func createImportMapTag(importMapping map[string]string, nonce string) string {

	importMap := "<script type=\"importmap\""
	if nonce != "" {
		importMap += fmt.Sprintf(" nonce=\"%s\"", nonce)
	}
	importMap += ">\n{\n\t\"imports\":{"

	for k, v := range importMapping {
		importMap += fmt.Sprintf("\n\t\t\"%s\":\"%s\",", k, v)
	}
	importMap = strings.TrimSuffix(importMap, ",")
	importMap += "\n\t}\n}\n</script>"

	return importMap
}

// AssetHandler serves assets from the local directory `assets/assets`.  Assets are loaded from this
// directory on start up and served from memory.  Any errors during start up will be served by AssetHandlers.
// Assets are served at the path `/assets/...` and can be also be served with a hashed path which finger prints the asset
// for uniqueness for caching e.g.,
//
//	/assets/bootstrap/hello.css
//	/assets/bootstrap/1fdd2266-hello.css
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
		a.fileType = suffix
	}

	// these types should appear in weft.compressibleMimes as appropriate
	switch suffix {
	case "js", "mjs", "map":
		a.mime = "text/javascript"
	case "css":
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
	case "svg":
		a.mime = "image/svg+xml"
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

	a.sri, err = calcSRIhash(a.b)
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
			assets[a.hashedPath] = a
			assets[a.path] = a
			assetHashes[a.path] = a.hashedPath
		}
	}

	return nil
}
