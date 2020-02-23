package cachedclient

import (
	"crypto/sha1"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/brinick/github/client"
	"github.com/brinick/logging"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			logging.Fatal(
				"Error os.Stat()-ing the cache file",
				logging.F("file", path),
			)
		}
		return false
	}
	return true
}

func createFileIfInexistant(path string) error {
	if !fileExists(path) {
		logging.Info("Creating cache file", logging.F("path", path))
		handler, err := os.Create(path)
		defer handler.Close()
		if err != nil {
			logging.Fatal("Unable to create cache file", logging.F("err", err))
			return err
		}
	}
	return nil
}

// ------------------------------------------------------------------

// CacheData represents the mapping of hash key
// to the corresponding payload
type CacheData map[string]client.Payload

// PickledCache represents a pickled cache
type PickledCache struct {
	Path string
	Data CacheData
}

var errInexistantCache = errors.New("Inexistant cache")

// NewCache will create and initialise a new PickledCached file
func NewCache() (*PickledCache, error) {
	cachePath, found := os.LookupEnv("GITHUB_CACHE_FILE")
	if !found {
		// default
		cachePath = ".github-cache"
	}
	cachePath, _ = filepath.Abs(cachePath)
	logging.Info("Using cache file", logging.F("path", cachePath))
	cache := &PickledCache{cachePath, nil}

	// If this is the first time we use this cache file
	// then it will not exist, which is ok (it will be created
	// when we save). If however a different error is found
	// that is not ok.
	if err := cache.Load(); err != nil && err != errInexistantCache {
		logging.Fatal("Failed to create a PickledCache", logging.F("err", err))
	}
	return cache, nil
}

func (pc *PickledCache) get(key string) (client.Payload, bool) {
	val, found := pc.Data[key]
	return val, found
}

// Load retieves the cache data from file
// and puts in the cache data member
func (pc *PickledCache) Load() error {
	if !fileExists(pc.Path) {
		pc.Data = map[string]client.Payload{}
		return errInexistantCache
	}

	handler, err := os.Open(pc.Path)
	if err != nil {
		logging.Error(
			"Unable to open cache file",
			logging.F("file", pc.Path),
			logging.F("err", err),
		)
		return err
	}
	defer handler.Close()

	//data := new(map[string]map[string]string)
	data := new(CacheData)
	decoder := gob.NewDecoder(handler)
	err = decoder.Decode(data)
	if err != nil {
		logging.Error("Unable to decode cache file data", logging.F("err", err))
	} else {
		// no error, we can update the cache data
		pc.Data = *data
		logging.Debug("Loaded cache file data")
	}
	return err
}

// Save dumps the in-memory cache data to file
func (pc PickledCache) Save() error {
	handler, err := os.Create(pc.Path)
	if err != nil {
		logging.Error(
			"Unable to open cache file for saving",
			logging.F("err", err),
		)
		return err
	}
	defer handler.Close()

	encoder := gob.NewEncoder(handler)
	err = encoder.Encode(&pc.Data)
	if err != nil {
		logging.Error(
			"Unable to save cache data",
			logging.F("err", err),
		)
	}
	return err
}

func (pc *PickledCache) update(d map[string]client.Payload, save bool) {
	for key, val := range d {
		pc.Data[key] = val
	}

	if save {
		pc.Save()
	}
}

func (pc *PickledCache) delete(key string) {
	// this is a no-op if the key is not present
	delete(pc.Data, key)
}

func (pc *PickledCache) generateCacheID(entries [][2]string) string {
	s := sha1.New()
	for _, entry := range entries {
		key, val := entry[0], entry[1]
		io.WriteString(s, string(key))
		io.WriteString(s, string(val))
	}
	return fmt.Sprintf("%x", s.Sum(nil))
}
