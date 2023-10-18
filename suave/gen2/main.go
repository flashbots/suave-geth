package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"text/template"

	"github.com/ethereum/go-ethereum/core/vm"
)

func main() {
	fmt.Println(vm.GetRuntime().GetMethods())

	for _, method := range vm.GetRuntime().GetMethods() {
		fmt.Println(method.Inputs)
	}

	input := map[string]interface{}{
		"Methods": vm.GetRuntime().GetMethods(),
	}
	if err := applyTemplate(suaveLibTemplate, input, "suave.sol"); err != nil {
		panic(err)
	}
}

func applyTemplate(templateText string, input interface{}, out string) error {
	funcMap := template.FuncMap{}

	t, err := template.New("template").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return err
	}

	var outputRaw bytes.Buffer
	if err = t.Execute(&outputRaw, input); err != nil {
		return err
	}

	// escape any quotes
	str := outputRaw.String()
	str = strings.Replace(str, "&#34;", "\"", -1)
	str = strings.Replace(str, "&amp;", "&", -1)
	str = strings.Replace(str, ", )", ")", -1)

	/*
		if str, err = formatSolidity(str); err != nil {
			return err
		}
	*/

	fmt.Println(str)
	return nil
}

var suaveLibTemplate = `// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

contract Suave {
	{{ range .Methods }}
	{{ . }}
	{{ end }}
}
`

func formatSolidity(code string) (string, error) {
	// Check if "forge" command is available in PATH
	_, err := exec.LookPath("forge")
	if err != nil {
		return "", fmt.Errorf("forge command not found in PATH: %v", err)
	}

	// Command and arguments for forge fmt
	command := "forge"
	args := []string{"fmt", "--raw", "-"}

	// Create a command to run the forge fmt command
	cmd := exec.Command(command, args...)

	// Set up input from stdin
	cmd.Stdin = bytes.NewBufferString(code)

	// Set up output buffer
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	// Run the command
	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("error running command: %v", err)
	}

	return outBuf.String(), nil
}
