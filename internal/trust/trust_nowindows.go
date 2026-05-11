//go:build !windows

package trust

func installWindows(string) error { return errWrongGOOS("windows") }

func uninstallWindows() error { return errWrongGOOS("windows") }
