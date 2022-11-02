package appcache

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"git.eth4.dev/golibs/filepaths"
)

const (
	cacheFileName = "cache.dat"
)

// ReadFromCacheStore - читает из кэша указанного приложения значение по ключу
func ReadFromCacheStore(appName, key string) interface{} {
	fd, err := os.OpenFile(
		filepath.Clean(filepaths.CachePath(appName, cacheFileName)),
		os.O_RDONLY,
		os.ModePerm,
	)
	if err != nil {
		return nil
	}

	buf := &bytes.Buffer{}

	if _, err = io.Copy(buf, fd); err != nil {
		return nil
	}

	_ = fd.Close()
	store := make(map[string]interface{})

	if err = json.NewDecoder(buf).Decode(&store); err != nil {
		return nil
	}

	return store[key]
}
