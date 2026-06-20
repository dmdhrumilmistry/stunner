// Command stunnerd is a headless harness for developing and testing the Stunner
// core without the Flutter UI. In the skeleton it prints the version and a
// freshly generated identity fingerprint; as roadmap phases land it will grow
// into a runnable node (transport + signaling) for two-instance testing.
package main

import (
	"fmt"
	"os"

	"github.com/dmdhrumilmistry/stunner/core/pkg/core"
)

func main() {
	fmt.Println(core.VersionString())

	fp, err := core.NewIdentityFingerprint()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to generate identity:", err)
		os.Exit(1)
	}
	fmt.Println("generated identity fingerprint:")
	fmt.Println("  " + fp)
}
