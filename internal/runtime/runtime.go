package runtime

const defaultLsLimit = 10

type Runtime interface {
	Name() string

	availableVersions(stableOnly bool) (map[string]string, error)
	installedVersions() ([]string, error)
	removeShims() error

	Ls(limit int) error
	Install(version string) error
	Uninstall(version string) error
	Active() (string, error)
	Use(version string) error
	// RemoveShims removes this runtime's shims (e.g. after purge when no version is active).
	RemoveShims() error

	// AvailableVersions is a public wrapper around availableVersions, used by the CLI
	// to resolve version strings before running commands.
	AvailableVersions(stableOnly bool) (map[string]string, error)

	// InstalledVersions is a public wrapper around installedVersions, used by the CLI
	// to resolve version strings before running commands.
	InstalledVersions() ([]string, error)
}
