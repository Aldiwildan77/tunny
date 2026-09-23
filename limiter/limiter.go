package limiter

type Limiter interface {
	Set(key string, limit int) error
	Acquire(key string) bool
	Release(key string)
	Remove(key string) error
}
