//go:build windows

package trust

func installWindows(caPEM string) error {
	return certutilAddRoot(caPEM)
}

func uninstallWindows() error {
	return certutilDelRoot()
}
