//go:build !linux

package trust

func installLinux(string) error { return errWrongGOOS("linux") }

func uninstallLinux() error { return errWrongGOOS("linux") }
