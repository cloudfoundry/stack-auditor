package terminalUI_test

import (
	"bufio"
	"bytes"

	"github.com/cloudfoundry/stack-auditor/terminalUI"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TerminalUI", func() {
	var (
		uiController terminalUI.UIController
		outputBuffer bytes.Buffer
		inputBuffer  bytes.Buffer
	)

	BeforeEach(func() {
		outputBuffer = bytes.Buffer{}
		outputWriter := bufio.NewWriter(&outputBuffer)

		inputBuffer = bytes.Buffer{}
		testReader := bufio.NewReader(&inputBuffer)
		testScanner := bufio.NewScanner(testReader)

		uiController = terminalUI.UIController{Scanner: testScanner, OutputWriter: outputWriter}

	})

	When("deleting a stack", func() {
		BeforeEach(func() {
			// clean up all of our streams
			outputBuffer.Reset()
			inputBuffer.Reset()
		})
		It("return true when user types yes", func() {
			inputBuffer.WriteString("some-stack-name")
			Expect(uiController.ConfirmDelete("some-stack-name")).To(BeTrue())
			Expect(outputBuffer.String()).To(ContainSubstring("Deleting stack"))
		})

		It("returns fals when no user input", func() {
			inputBuffer.WriteString("")
			Expect(uiController.ConfirmDelete("some-stack-name")).To(BeFalse())
			Expect(outputBuffer.String()).To(ContainSubstring("failed to scan user input aborting"))
		})

		It("returns fals when user types something other than yes", func() {
			inputBuffer.WriteString("some-stack-name-that-is-totes-mcgoats-wrong")
			Expect(uiController.ConfirmDelete("some-stack-name")).To(BeFalse())
			Expect(outputBuffer.String()).To(ContainSubstring("aborted deleting stack"))
		})

	})
})
