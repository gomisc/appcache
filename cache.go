package appcache

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"git.eth4.dev/golibs/errors"
	"git.eth4.dev/golibs/filepaths"
	"git.eth4.dev/golibs/iorw"
)

const (
// CacheFileName - имя файла кэша

)

// AppCache - кеш хранения данных приложения
type AppCache interface {
	io.Closer
	Read(key string) interface{}
	Write(key string, value interface{})
}

type appCache struct {
	fd      *os.File
	bufPool *BufPool
	opts    *cacheOptions
	cancel  context.CancelFunc

	mu    sync.RWMutex
	store map[string]interface{}
}

// Open - конструктор кэша приложения
func Open(appName string, options ...Option) (AppCache, error) {
	opts := processOptions(options...)

	cacheDir := filepaths.CachePath(appName)
	if !filepaths.FileExists(cacheDir) {
		if err := iorw.MakeDirs(cacheDir); err != nil {
			return nil, errors.Wrap(err, "create cache dir")
		}
	}

	fd, err := os.OpenFile(filepath.Clean(filepath.Join(cacheDir, cacheFileName)), os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return nil, errors.Wrap(err, "open cache file storage")
	}

	cache := &appCache{
		fd:      fd,
		opts:    opts,
		bufPool: NewBuffPool(runtime.NumCPU()),
	}

	if err = cache.load(); err != nil {
		return nil, errors.Wrap(err, "load cache")
	}

	if opts.storeDumpInterval != 0 {
		ctx, cancel := context.WithCancel(context.Background())
		cache.cancel = cancel

		go cache.saveTimer(ctx)
	}

	return cache, nil
}

func (c *appCache) Read(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.store[key]
}

func (c *appCache) Write(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[key] = value
}

// Close - реализация io.Closer
func (c *appCache) Close() error {
	var retErr error

	if c.cancel != nil {
		c.cancel()
	}

	c.mu.Lock()

	if err := c.save(); err != nil {
		retErr = errors.And(retErr, errors.Wrap(err, "save cache to file"))
	}

	if err := c.fd.Close(); err != nil {
		retErr = errors.And(retErr, errors.Wrap(err, "close cache file"))
	}

	c.mu.Unlock()

	return retErr
}

func (c *appCache) load() error {
	buf := c.bufPool.Get()

	defer func() {
		buf.Reset()
		c.bufPool.Put(buf)
	}()

	if _, err := io.Copy(buf, c.fd); err != nil {
		return errors.Wrap(err, "read cache file")
	}

	if err := json.NewDecoder(buf).Decode(&c.store); err != nil {
		if !errors.Is(err, io.EOF) {
			return errors.Wrap(err, "decode cache data")
		}

		c.store = make(map[string]interface{})
	}

	return nil
}

func (c *appCache) truncate() error {
	if err := c.fd.Truncate(0); err != nil {
		return errors.Wrap(err, "truncate cache file")
	}

	if _, err := c.fd.Seek(0, 0); err != nil {
		return errors.Wrap(err, "seek cache file")
	}

	return nil
}

func (c *appCache) save() error {
	buf := c.bufPool.Get()

	defer func() {
		buf.Reset()
		c.bufPool.Put(buf)
	}()

	if err := c.truncate(); err != nil {
		return errors.Wrap(err, "truncate")
	}

	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(c.store); err != nil {
		return errors.Wrap(err, "encode cache store")
	}

	if err := buf.WriteByte('\n'); err != nil {
		return errors.Wrap(err, "buffering cache records")
	}

	if _, err := buf.WriteTo(c.fd); err != nil {
		return errors.Wrap(err, "write buffer to file")
	}

	return nil
}

func (c *appCache) saveTimer(ctx context.Context) {
	if c.opts.storeDumpInterval == 0 {
		return
	}

	ticker := time.NewTicker(c.opts.storeDumpInterval)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			_ = c.save()
			c.mu.RUnlock()
		}
	}
}
