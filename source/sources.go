package source

type Factory func(url string) (Source, error)

type name struct {
}

func Register(prefix string, factory Factory) {

}
