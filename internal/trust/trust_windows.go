//go:build windows

package trust

func installWindows(caPEM string) error {
	// Current-user root store (no admin).
	return run("certutil", "-user", "-addstore", "Root", caPEM)
}

func uninstallWindows() error {
	return run("certutil", "-user", "-delstore", "Root", "Buick Development CA")
}
