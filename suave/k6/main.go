package main

import (
	"fmt"

	k6cmd "go.k6.io/k6/cmd"
	"go.k6.io/k6/js/modules"
)

func main() {
	k6cmd.Execute()
}

// init is called by the Go runtime at application startup.
func init() {
	modules.Register("k6/x/compare", new(Compare))
}

// Compare is the type for our custom API.
type Compare struct {
	ComparisonResult string // textual description of the most recent comparison
}

// IsGreater returns true if a is greater than b, or false otherwise, setting textual result message.
func (c *Compare) IsGreater(a, b int) bool {
	if a > b {
		c.ComparisonResult = fmt.Sprintf("%d is greater than %d", a, b)
		return true
	} else {
		c.ComparisonResult = fmt.Sprintf("%d is NOT greater than %d", a, b)
		return false
	}
}
