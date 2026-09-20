// Command professional-email-drafting drafts a professional email from supplied
// facts and refuses to send anything without a recorded human approval.
//
// The pipeline is not wired yet. This entry point is a placeholder that keeps the
// module building offline while the standalone port lands its packages; the real
// CLI replaces it.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "professional-email-drafting: pipeline not wired yet")
	os.Exit(2)
}
