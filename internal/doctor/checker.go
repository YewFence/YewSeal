package doctor

import "fmt"

// RequiredTools is empty — age and sops are embedded as Go libraries.
var RequiredTools = []string{}

// CheckTools verifies that all required external tools are installed.
// With age/sops embedded, this always returns nil.
func CheckTools() error {
	return nil
}

// CheckToolsVerbose displays embedded library versions.
func CheckToolsVerbose() bool {
	fmt.Println("Embedded libraries (no installation needed):")
	fmt.Println("  ✓ age: filippo.io/age (embedded)")
	fmt.Println("  ✓ sops: github.com/YewFence/sops/v3 (embedded, native TOML support)")

	fmt.Println()
	fmt.Println("All dependencies are embedded. No external tools are required.")
	return true
}
