package source

import (
	"errors"
	"github.com/zgwit/iot-master/v4/lib"
	"strings"
)

type Factory func(url string, options Options) (Source, error)

var factories = map[string]Factory{}

// var sources = map[string]Source{}
var sources lib.Map[Source]

func Register(prefix string, factory Factory) {
	factories[prefix] = factory
}

func Create(url string, options Options) (source Source, err error) {
	urls := strings.Split(url, "://")
	prefix := urls[0]
	if factory, ok := factories[prefix]; ok {
		source, err = factory(url, options)
	} else {
		err = errors.New("不支持的模式")
	}
	return
}

func Get(url string, options Options) (source Source, err error) {
	src := sources.Load(url)
	if src == nil {
		source, err = Create(url, options)
		if err != nil {
			return
		}
		sources.Store(url, &source)
	} else {
		source = *src
	}

	return source, source.Check()
}
