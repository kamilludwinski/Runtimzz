package cmd

import (
	"fmt"

	"github.com/kamilludwinski/runtimzzz/internal/logger"
	"github.com/kamilludwinski/runtimzzz/internal/msgs"
	"github.com/kamilludwinski/runtimzzz/internal/output"
	"github.com/kamilludwinski/runtimzzz/internal/runtime"
	"github.com/kamilludwinski/runtimzzz/internal/utils/versionutils"
)

// runtime subcommands that require a version argument
const (
	cmdLs           = "ls"
	cmdInstall      = "install"
	cmdInstallShort = "i"
	cmdUninstall    = "uninstall"
	cmdUninstallShort = "u"
	cmdUse            = "use"
	cmdPurge          = "purge"
)

func HandleRuntime(args []string) {
	logger.Debug("HandleRuntime", "args", args)

	if len(args) < 2 {
		printHelp()
		return
	}

	name := args[0]
	subcmd := args[1]

	rt, err := runtime.Get(name)
	if err != nil {
		logger.Debug("runtime not found", "name", name, "err", err)
		output.Error(err.Error())
		return
	}

	needVersion := subcmd == cmdInstall || subcmd == cmdInstallShort ||
		subcmd == cmdUninstall || subcmd == cmdUninstallShort || subcmd == cmdUse
	if needVersion && len(args) < 3 {
		output.Warn(msgs.MissingVersion(name, subcmd))
		return
	}

	switch subcmd {
	case cmdLs:
		if err := rt.Ls(0); err != nil {
			logger.Error("ls failed", "err", err)
			output.Error(err.Error())
		}
	case cmdInstall, cmdInstallShort:
		requested := args[2]

		availableMap, err := rt.AvailableVersions(true)
		if err != nil {
			logger.Error("install: failed to get available versions", "err", err)
			output.Error(fmt.Sprintf("Failed to get available versions for %s: %v", name, err))
			return
		}
		var available []string
		for v := range availableMap {
			available = append(available, v)
		}

		resolved := requested
		if r, ok := versionutils.ResolveVersion(requested, available); ok {
			resolved = r
			if resolved != requested {
				logger.Debug("resolved install version", "input", requested, "resolved", resolved)
				output.Info(msgs.ResolvedVersion(requested, resolved))
			}
		}

		if err := rt.Install(resolved); err != nil {
			logger.Error("install failed", "version", resolved, "err", err)
			output.Error(err.Error())
		}
	case cmdUninstall, cmdUninstallShort:
		if err := rt.Uninstall(args[2]); err != nil {
			logger.Error("uninstall failed", "version", args[2], "err", err)
			output.Error(err.Error())
		}
	case cmdUse:
		requested := args[2]

		installed, err := rt.InstalledVersions()
		if err != nil {
			logger.Error("use: failed to get installed versions", "err", err)
			output.Error(fmt.Sprintf("Failed to get installed versions for %s: %v", name, err))
			return
		}

		resolved := requested
		if r, ok := versionutils.ResolveVersion(requested, installed); ok {
			resolved = r
			if resolved != requested {
				logger.Debug("resolved use version", "input", requested, "resolved", resolved)
				output.Info(msgs.ResolvedVersion(requested, resolved))
			}
		}

		if err := rt.Use(resolved); err != nil {
			logger.Error("use failed", "version", resolved, "err", err)
			output.Error(err.Error())
		}
	case cmdPurge:
		versions, err := runtime.ListInstalledVersions(name)
		if err != nil {
			logger.Error("purge failed: list installed versions", "runtime", name, "err", err)
			output.Error(fmt.Sprintf("Failed to list installed versions for %s: %v", name, err))
			return
		}
		if len(versions) == 0 {
			output.Warn(fmt.Sprintf("No %s versions installed; nothing to purge", name))
			return
		}

		output.Info(fmt.Sprintf("Purging all %s versions...", name))
		for _, v := range versions {
			if err := rt.Uninstall(v); err != nil {
				logger.Error("purge uninstall failed", "runtime", name, "version", v, "err", err)
				output.Error(fmt.Sprintf("Failed to uninstall %s %s: %v", name, v, err))
				return
			}
		}
		if err := rt.RemoveShims(); err != nil {
			logger.Error("purge: remove shims failed", "runtime", name, "err", err)
			output.Error(fmt.Sprintf("Failed to remove %s shims: %v", name, err))
			return
		}
		if runState != nil {
			if err := runState.ClearRuntime(name); err != nil {
				logger.Error("purge: clear state failed", "runtime", name, "err", err)
				output.Error(fmt.Sprintf("Failed to clear %s state: %v", name, err))
				return
			}
		}
		output.Success(fmt.Sprintf("All %s versions purged", name))
	default:
		logger.Debug("unknown command", "cmd", subcmd)
		output.Error(msgs.UnknownCommand(subcmd))
	}
}
