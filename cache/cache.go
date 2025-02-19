package cache

type Cache interface {
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	Delete(key string)
}

var PreviewCache Cache = NewMemoryCache()
