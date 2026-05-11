package trust

import "fmt"

func errWrongGOOS(want string) error {
	return fmt.Errorf("trust: internal build tag mismatch (expected %s)", want)
}
