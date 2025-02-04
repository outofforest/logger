package remote

// Config stores configuration of remote logger.
type Config[T comparable] struct {
	URL      string
	User     string
	Password string
	Labels   T
}
