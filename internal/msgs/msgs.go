package msgs

import "fmt"

func Installing(displayName, version string) string {
	return fmt.Sprintf("%s %s is being installed...", displayName, version)
}

func Installed(displayName, version string) string {
	return fmt.Sprintf("%s %s installed", displayName, version)
}

func Uninstalling(displayName, version string) string {
	return fmt.Sprintf("%s %s is being uninstalled...", displayName, version)
}

func Uninstalled(displayName, version string) string {
	return fmt.Sprintf("%s %s uninstalled", displayName, version)
}

func Activating(displayName, version string) string {
	return fmt.Sprintf("Activating %s version %s...", displayName, version)
}

func ActiveSet(displayName, version string) string {
	return fmt.Sprintf("%s version %s is now active", displayName, version)
}

func AlreadyActive(displayName, version string) string {
	return fmt.Sprintf("%s version %s is already active", displayName, version)
}

func LsHeader(displayName string, available, installed int) string {
	return fmt.Sprintf("%s: %d available, %d installed", displayName, available, installed)
}

func UseHint(runtime, version string) string {
	return fmt.Sprintf("Run 'rtz %s use %s' to activate.", runtime, version)
}

func MissingVersion(runtime, subcmd string) string {
	return fmt.Sprintf("Missing version (e.g. rtz %s %s <version>)", runtime, subcmd)
}

func UnknownCommand(cmd string) string {
	return "Unknown command: " + cmd
}

func NoActiveVersion(runtime string) string {
	return fmt.Sprintf("No active %s version set (run: rtz %s use <version>)", runtime, runtime)
}

func ResolvedVersion(input, resolved string) string {
	return fmt.Sprintf("Resolved version %s to %s", input, resolved)
}
