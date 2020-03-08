package goroutine

import (
	exposedRuntime "github.com/huandu/go-tls/g"
)

type g struct{}

func GetG() *g {
	return (*g)(exposedRuntime.G())
}
