package gorex

import (
	exposedRuntime "github.com/huandu/go-tls/g"
)

// G is just a placeholder. It was supposed to be an alias to runtime.g.
type G struct{}

// GetG is an alias to runtime.getg
func GetG() *G {
	return (*G)(exposedRuntime.G())
}
