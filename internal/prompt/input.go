package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Session owns buffered input and prompt delivery for one interaction.
// Once delivery fails, it never consumes another answer.
type Session struct {
	reader *bufio.Reader
	writer io.Writer
	err    error
}

func New(reader io.Reader, writer io.Writer) *Session {
	return &Session{reader: bufio.NewReader(reader), writer: writer}
}

func (s *Session) Err() error { return s.err }

func (s *Session) ask(text string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	n, err := io.WriteString(s.writer, text)
	if err == nil && n != len(text) {
		err = io.ErrShortWrite
	}
	if err != nil {
		s.err = fmt.Errorf("failed to write prompt: %w", err)
		return "", s.err
	}
	return s.reader.ReadString('\n')
}

func (s *Session) PromptWithDefault(prompt, fallback string) string {
	input, err := s.ask(fmt.Sprintf("%s [%s]: ", prompt, fallback))
	if err != nil {
		if s.err != nil {
			return ""
		}
		return fallback
	}
	if input = strings.TrimSpace(input); input == "" {
		return fallback
	}
	return input
}

func (s *Session) PromptRequired(prompt string) (string, error) {
	for {
		input, err := s.ask(prompt + ": ")
		input = strings.TrimSpace(input)
		if input != "" {
			return input, nil
		}
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
	}
}

func (s *Session) PromptOptional(prompt string) string {
	input, err := s.ask(prompt + ": ")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(input)
}

func (s *Session) PromptYesNo(prompt string, defaultYes bool) bool {
	suffix := "[y/N]"
	if defaultYes {
		suffix = "[Y/n]"
	}
	input, err := s.ask(fmt.Sprintf("%s %s: ", prompt, suffix))
	if err != nil {
		if s.err != nil {
			return false
		}
		return defaultYes
	}
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

func (s *Session) PromptYesNoConditional(flagSet, defaultValue bool, prompt string) bool {
	if flagSet {
		return defaultValue
	}
	return s.PromptYesNo(prompt, defaultValue)
}

// Check combines interaction delivery failure with any independently found error.
func (s *Session) Check(err error) error {
	if errors.Is(err, s.err) {
		return err
	}
	return errors.Join(err, s.err)
}
