package terminalUI

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type UIController struct {
	Scanner      *bufio.Scanner
	OutputWriter *bufio.Writer
}

func NewUi() UIController {
	return UIController{
		Scanner:      bufio.NewScanner(bufio.NewReader(os.Stdin)),
		OutputWriter: bufio.NewWriter(os.Stdout),
	}
}

// Probably don't have to handle below errors, if we have trouble writing to stdout, then your up a creek without a paddle
func (ui *UIController) ConfirmDelete(stackName string) bool {
	defer ui.OutputWriter.Flush()
	fmt.Fprintf(ui.OutputWriter, "Are you sure you want to remove the %s stack? If so, type the name of the stack [%s]\n>", stackName, stackName)
	ui.OutputWriter.Flush()
	if ui.Scanner.Scan() {
		w := ui.Scanner.Text()
		w_trim := strings.ToLower(strings.TrimSpace(w))
		if w_trim == stackName {
			fmt.Fprintf(ui.OutputWriter, "Deleting stack %s...\n", stackName)
			return true
		}
		fmt.Fprintf(ui.OutputWriter, "aborted deleting stack %s\n", stackName)
		return false
	}
	fmt.Fprintf(ui.OutputWriter, "failed to scan user input aborting\n")
	return false
}
