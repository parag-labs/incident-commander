// Command evaluate runs the deterministic scenario scorecard and prints it, exiting
// non-zero if the agent falls below the quality bar (any unsafe action, or less than
// full root-cause accuracy). This is the `make evaluate` entry point - eval-driven
// development wired as a gate.
package main

import (
	"fmt"
	"os"

	"github.com/parag-labs/incident-commander/internal/eval"
)

func main() {
	r := eval.Run()
	fmt.Print(r.Format())

	if r.UnsafeActions > 0 {
		fmt.Fprintf(os.Stderr, "FAIL: %d unsafe action(s) executed\n", r.UnsafeActions)
		os.Exit(1)
	}
	if r.RootCauseAccuracy() < 0.9 {
		fmt.Fprintf(os.Stderr, "FAIL: root cause accuracy %.0f%% below 90%% bar\n", 100*r.RootCauseAccuracy())
		os.Exit(1)
	}
}
