//go:build darwin

package platform

import "fmt"

type PlatformImpl struct{}

var _ Platform = (*PlatformImpl)(nil)

func (p PlatformImpl) GoArchiveExt() string {
	return "tar.gz"
}

func (p PlatformImpl) EnsurePath(path string) error {
	return nil
}

func (p PlatformImpl) CreateGoShims(version string) error {
	return fmt.Errorf("shims are not supported on this platform (Windows only for now)")
}

func (p PlatformImpl) CreateShims(binDir string, tools []string) error {
	return fmt.Errorf("shims are not supported on this platform (Windows only for now)")
}
