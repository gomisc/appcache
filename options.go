package appcache

import (
	"time"
)

type cacheOptions struct {
	storeDumpInterval time.Duration
}

// Option - опция кэша приложения
type Option func(o *cacheOptions)

// SaveInterval - интервал периодического сохранения кэша
func SaveInterval(t time.Duration) Option {
	return func(o *cacheOptions) {
		o.storeDumpInterval = t
	}
}

func processOptions(o ...Option) *cacheOptions {
	opts := &cacheOptions{}

	for i := 0; i < len(o); i++ {
		o[i](opts)
	}

	return opts
}
