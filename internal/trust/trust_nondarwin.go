//go:build !darwin

package trust

func installDarwin(string) error { return errWrongGOOS("darwin") }

func uninstallDarwin() error { return errWrongGOOS("darwin") }
