package utils

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
}

func (c Command) Run(bin, dir string, quiet bool, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	if quiet {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func (c Command) RunWithOutput(bin, dir string, quiet bool, args ...string) (string, error) {
	logs := &bytes.Buffer{}

	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	if quiet {
		cmd.Stdout = io.MultiWriter(io.Discard, logs)
		cmd.Stderr = io.MultiWriter(io.Discard, logs)
	} else {
		cmd.Stdout = io.MultiWriter(os.Stdout, logs)
		cmd.Stderr = io.MultiWriter(os.Stderr, logs)
	}
	err := cmd.Run()

	return strings.TrimSpace(logs.String()), err
}

func (c Command) SetEnv(variableName string, path string) error {
	return os.Setenv(variableName, path)
}
