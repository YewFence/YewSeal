package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var (
	cachedStdin       *os.File
	cachedStdinReader *bufio.Reader
)

func stdinReader() *bufio.Reader {
	if cachedStdin != os.Stdin || cachedStdinReader == nil {
		cachedStdin = os.Stdin
		cachedStdinReader = bufio.NewReader(os.Stdin)
	}
	return cachedStdinReader
}

// PromptWithDefault prompts user for input with a default value
// If the user presses Enter without typing anything, returns the default value
func PromptWithDefault(prompt, defaultValue string) string {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	input, err := stdinReader().ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

// PromptRequired prompts user until a non-empty value is entered.
func PromptRequired(prompt string) string {
	for {
		fmt.Printf("%s: ", prompt)
		input, err := stdinReader().ReadString('\n')
		if err != nil {
			continue
		}

		input = strings.TrimSpace(input)
		if input != "" {
			return input
		}

		fmt.Println("⚠️  Value cannot be empty.")
	}
}

// PromptYesNo prompts user for a yes/no confirmation
// defaultYes controls the default behavior when user presses Enter
// Returns true for yes, false for no
func PromptYesNo(prompt string, defaultYes bool) bool {
	var suffix string
	if defaultYes {
		suffix = "[Y/n]"
	} else {
		suffix = "[y/N]"
	}

	fmt.Printf("%s %s: ", prompt, suffix)
	input, err := stdinReader().ReadString('\n')
	if err != nil {
		return defaultYes
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}

	return input == "y" || input == "yes"
}

// PromptYesNoConditional conditionally prompts user based on whether a flag is set
// If flagSet is true, returns defaultValue without prompting
// If flagSet is false, prompts the user and returns their answer
func PromptYesNoConditional(flagSet bool, defaultValue bool, prompt string) bool {
	if flagSet {
		return defaultValue
	}
	return PromptYesNo(prompt, defaultValue)
}
