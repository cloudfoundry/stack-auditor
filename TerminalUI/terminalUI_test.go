package TerminalUI

import (
	"bufio"
	"bytes"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"testing"
)

func TestUnitTerminalUI(t *testing.T) {
	spec.Run(t, "Deleter", testTerminalUI, spec.Report(report.Terminal{}))
}

func testTerminalUI(t *testing.T, when spec.G, it spec.S) {
	var (
		uiController UIController
		outputBuffer bytes.Buffer
		inputBuffer  bytes.Buffer
	)

	it.Before(func() {
		RegisterTestingT(t)

		outputBuffer = bytes.Buffer{}
		outputWriter := bufio.NewWriter(&outputBuffer)

		inputBuffer = bytes.Buffer{}
		testReader := bufio.NewReader(&inputBuffer)
		testScanner := bufio.NewScanner(testReader)

		uiController = UIController{scanner: testScanner, outputWriter: outputWriter}

	})

	when("deleting a stack", func() {
		it.Before(func() {
			// clean up all of our streams
			outputBuffer.Reset()
			inputBuffer.Reset()
		})
		it("return true when user types yes", func() {
			inputBuffer.WriteString("some-stack-name")
			Expect(uiController.ConfirmDelete("some-stack-name")).To(BeTrue())
			Expect(outputBuffer.String()).To(ContainSubstring("Deleting stack"))
		})

		it("returns fals when no user input", func() {
			inputBuffer.WriteString("")
			Expect(uiController.ConfirmDelete("some-stack-name")).To(BeFalse())
			Expect(outputBuffer.String()).To(ContainSubstring("failed to scan user input aborting"))
		})

		it("returns fals when user types something other than yes", func() {
			inputBuffer.WriteString("some-stack-name-that-is-totes-mcgoats-wrong")
			Expect(uiController.ConfirmDelete("some-stack-name")).To(BeFalse())
			Expect(outputBuffer.String()).To(ContainSubstring("aborted deleting stack"))
		})

	})
}
