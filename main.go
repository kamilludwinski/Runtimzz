package main

import (
	"os"

	
	"github.com/kamilludwinski/runtimzzz/cmd"
	"github.com/kamilludwinski/runtimzzz/internal/logger"
	"github.com/kamilludwinski/runtimzzz/internal/output"
	"github.com/kamilludwinski/runtimzzz/internal/runtime"
	"github.com/kamilludwinski/runtimzzz/internal/state"
	"github.com/kamilludwinski/runtimzzz/internal/update"
)

func main() {
	if update.RunUpdaterIfRequested() {
		return
	}

	logger.Init()
	logger.Debug("main started", "args", os.Args)

	state := state.NewState()
	if err := state.Load(); err != nil {
		logger.Error("failed to load state", "err", err)
		output.Error("Failed to load state: " + err.Error())
		return
	}

	logger.Debug("state loaded")

	goRuntime := runtime.Init(state)
	runtime.Register(goRuntime)

	nodeRuntime := runtime.InitNode(state)
	runtime.Register(nodeRuntime)

	pythonRuntime := runtime.InitPython(state)
	runtime.Register(pythonRuntime)

	logger.Debug("runtimes registered")

	cmd.Run(os.Args, state)
}
