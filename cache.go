// Copyright 2018 Adam Tauber
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package colly

import (
	"crypto/sha1"
	"encoding/gob"
	"encoding/hex"
	"net/http"
	"os"
	"path"
)

// CacheHandleFunc returns a Response via cache.
type CacheHandleFunc func(*http.Request, func() (*Response, error)) (*Response, error)

// CacheHandler is interface of CacheHandler.
type CacheHandler interface {
	// CacheHandle returns a Response via cache.
	CacheHandle(*http.Request, func() (*Response, error)) (*Response, error)
}

// NewCacheHandler returns a new CacheHandler.
type NewCacheHandler func(c *Collector) CacheHandler

// NewDefaultCacheHandler returns a new defaultCacheHandler.
func NewDefaultCacheHandler(c *Collector) CacheHandler {
	return &defaultCacheHandler{c}
}

type defaultCacheHandler struct {
	c *Collector
}

func (h *defaultCacheHandler) CacheHandle(
	request *http.Request,
	getter func() (*Response, error),
) (*Response, error) {
	if h.c.CacheDir == "" || request.Method != "GET" {
		return getter()
	}
	sum := sha1.Sum([]byte(request.URL.String()))
	hash := hex.EncodeToString(sum[:])
	dir := path.Join(h.c.CacheDir, hash[:2])
	filename := path.Join(dir, hash)
	if file, err := os.Open(filename); err == nil {
		resp := new(Response)
		err := gob.NewDecoder(file).Decode(resp)
		file.Close()
		if resp.StatusCode < 500 {
			return resp, err
		}
	}

	resp, err := getter()
	if err != nil {
		return resp, err
	}

	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return resp, err
		}
	}
	file, err := os.Create(filename + "~")
	if err != nil {
		return resp, err
	}
	if err := gob.NewEncoder(file).Encode(resp); err != nil {
		file.Close()
		return resp, err
	}
	file.Close()
	return resp, os.Rename(filename+"~", filename)
}
