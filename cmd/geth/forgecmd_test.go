package main

import (
	"fmt"
	"testing"
)

func TestForgeReadConfig(t *testing.T) {
	fmt.Println(app.Run([]string{"geth", "forge", "--local", "--config", "./testdata/forge.toml", "confidentialInputs"}))
}
