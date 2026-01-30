package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// PromptWithDefault prompts user for input with a default value
// If the user presses Enter without typing anything, returns the default value
func PromptWithDefault(prompt, defaultValue string) string {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}
	
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
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
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}
	
	return input == "y" || input == "yes"
}
