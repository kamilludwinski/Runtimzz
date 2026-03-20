package runtime

import (
	"fmt"

	"github.com/kamilludwinski/runtimzzz/internal/logger"
)

var runtimes = map[string]Runtime{}

func Register(rt Runtime) {
	name := rt.Name()
	logger.Debug("Register runtime", "name", name)
	runtimes[name] = rt
}

func Get(name string) (Runtime, error) {
	logger.Debug("Get runtime", "name", name)
	rt, ok := runtimes[name]
	if !ok {
		return nil, fmt.Errorf("runtime not found: %s", name)
	}

	return rt, nil
}
