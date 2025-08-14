// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

// Package assetembed provides a HTTP handler for serving asset files embedded in a Go binary through the embed.FS type.
// It is similar in purpose to http.FileServerFS() from the standard library,
// but instead of serving files directly with their names as found in the filesystem,
// it inserts a cryptographic digest of the file contents into the filename that the handler serves.
//
// This implements the "cache-busting pattern":
// User agents will immediately know when to update assets served by the HTTP server
// because links to those assets will change to refer to a new filename (that includes the hash of the updated file contents).
// This allows the HTTP handler to serve files with the very efficient "Cache-Control: immutable" caching method.
package assetembed

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	. "github.com/majewsky/gg/option"
)

// Handler serves static asset files, as described in the package documentation.
type Handler struct {
	digestPaths   map[string]string // e.g. "res/app.css" -> "res/app-OLBgp1GsljhM2TJ-sbHjaiH9txEUvgdDTAzHv2P24donTt6_529l-9Ua0vFImLlb.css"
	sriAttributes map[string]string // e.g. "res/app.css" -> "sha384-OLBgp1GsljhM2TJ+sbHjaiH9txEUvgdDTAzHv2P24donTt6/529l+9Ua0vFImLlb"
	contents      map[string][]byte // e.g. "res/app-e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855.css" -> [contents of res/app.css]
}

// NewHandler builds a new Handler instance.
// This will read all files in assetFS and store a copy of their contents inside the Handler instance.
//
// For filesystems backed by actual disk or network storage, this can be a very expensive operation.
// We recommend only using this function with embed.FS or other in-memory filesystems, for which the ReadFile() function is a cheap copy-by-reference.
func NewHandler(assetFS fs.ReadFileFS) (*Handler, error) {
	h := &Handler{
		digestPaths:   make(map[string]string),
		sriAttributes: make(map[string]string),
		contents:      make(map[string][]byte),
	}
	return h, fs.WalkDir(assetFS, ".", func(fullPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		buf, err := assetFS.ReadFile(fullPath)
		if err != nil {
			return err
		}
		fullPathWithDigest, sriAttribute := deriveAttributesForFile(fullPath, buf)
		h.digestPaths[fullPath] = fullPathWithDigest
		h.sriAttributes[fullPath] = sriAttribute
		h.contents[fullPathWithDigest] = buf
		return nil
	})
}

func deriveAttributesForFile(fullPath string, contents []byte) (pathWithDigest, sriAttribute string) {
	// we're using SHA512-384 here because that appears to be the common choice for SRI hashes
	digest := sha512.Sum384(contents)
	digestURLBase64 := base64.URLEncoding.EncodeToString(digest[:])
	digestStdBase64 := base64.StdEncoding.EncodeToString(digest[:])

	dirPath, fileName := path.Split(fullPath) // e.g. "res/", "app.css"
	ext := path.Ext(fileName)                 // e.g. ".css"
	fileNameWithDigest := fmt.Sprintf("%s-%s%s",
		strings.TrimSuffix(fileName, ext),
		digestURLBase64,
		ext,
	)
	return path.Join(dirPath, fileNameWithDigest), "sha384-" + digestStdBase64
}

// AssetPath takes the path to a file within the original filesystem,
// and returns the path with digest that the HTTP handler can serve.
// If the provided file path does not exist in the filesystem, None is returned.
//
// This function is intended for embedding URLs linking to assets served by this handler e.g. into HTML documents as part of templating.
//
// For example, if "res/app.css" is given and exists in the filesystem, then something like
// "res/app-OLBgp1GsljhM2TJ-sbHjaiH9txEUvgdDTAzHv2P24donTt6_529l-9Ua0vFImLlb.css" may be returned.
//
// A stability guarantee is given for the general shape of the transformed file path
// (only the basename is altered and the extension stays intact), but not for which digest is embedded and how.
func (h *Handler) AssetPath(originalPath string) Option[string] {
	result, exists := h.digestPaths[originalPath]
	if exists {
		return Some(result)
	} else {
		return None[string]()
	}
}

// SubresourceIntegrity takes the path to a file within the original filesystem,
// and returns a suitable value for the "integrity" attribute of an HTML element that supports Subresource Integrity.
//
// This method is intended to be used alongside AssetPath() during HTML template rendering.
//
// No stability guarantee is given for which digest is used to generated the result value.
func (h *Handler) SubresourceIntegrity(originalPath string) Option[string] {
	result, exists := h.sriAttributes[originalPath]
	if exists {
		return Some(result)
	} else {
		return None[string]()
	}
}

var cacheControlHeader = fmt.Sprintf(
	"public, max-age=%d, immutable",
	int64((14 * 24 * time.Hour).Seconds()), // max-age = 2 weeks
)

// ServeHTTP implements the http.Handler interface.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	buf, exists := h.contents[strings.TrimPrefix(r.URL.Path, "/")]
	if !exists {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodHead && r.Method != http.MethodGet {
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := mime.TypeByExtension(path.Ext(r.URL.Path))
	if contentType == "" {
		contentType = http.DetectContentType(buf)
	}

	w.Header().Set("Cache-Control", cacheControlHeader)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		w.Write(buf) //nolint:errcheck // no good way to deal with the error return
	}
}
