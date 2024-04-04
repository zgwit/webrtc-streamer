package source

import (
	"errors"
	"github.com/zgwit/iot-master/v4/lib"
	"github.com/zgwit/iot-master/v4/types"
	"strings"
)

type Factory func(url string, options types.Options) (Source, error)

var factories = map[string]Factory{}

// var sources = map[string]Source{}
var sources lib.Map[Source]

func Register(prefix string, factory Factory) {
	factories[prefix] = factory
}

func Create(url string, options types.Options) (source Source, err error) {
	urls := strings.Split(url, "://")
	prefix := urls[0]
	if factory, ok := factories[prefix]; ok {
		source, err = factory(url, options)
	} else {
		err = errors.New("不支持的模式")
	}
	return
}

func Get(url string, options types.Options) (source Source, err error) {
	source = *sources.Load(url)
	if source == nil {
		source, err = Create(url, options)
		if err != nil {
			return
		}
		sources.Store(url, &source)
	}

	return source, source.Check()
}
