package cache

type Cache interface {
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	Delete(key string)
}

/* public */
var SiteDataStore Cache = NewMemoryCache()

/* private */
var PreviewCache Cache = NewMemoryCache()
var UserDataStore Cache = NewMemoryCache()
