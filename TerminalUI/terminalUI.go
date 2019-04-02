package TerminalUI

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type UIController struct {
	scanner      *bufio.Scanner
	outputWriter *bufio.Writer
}

func NewUi() UIController {
	return UIController{
		scanner:      bufio.NewScanner(bufio.NewReader(os.Stdin)),
		outputWriter: bufio.NewWriter(os.Stdout),
	}
}

// Probably don't have to handle below errors, if we have trouble writing to stdout, then your up a creek without a paddle
func (ui *UIController) ConfirmDelete(stackName string) bool {
	defer ui.outputWriter.Flush()
	fmt.Fprintf(ui.outputWriter, "Are you sure you want to remove the %s stack? If so, type the name of the stack [%s]\n>", stackName, stackName)
	ui.outputWriter.Flush()
	if ui.scanner.Scan() {
		w := ui.scanner.Text()
		w_trim := strings.ToLower(strings.TrimSpace(w))
		if w_trim == stackName {
			fmt.Fprintf(ui.outputWriter, "Deleting stack %s...\n", stackName)
			return true
		}
		fmt.Fprintf(ui.outputWriter, "aborted deleting stack %s\n", stackName)
		return false
	}
	fmt.Fprintf(ui.outputWriter, "failed to scan user input aborting\n")
	return false
}
