package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

var builtinCMDs = []string{
	"exit",
	"echo",
	"type",
	"pwd",
	"cd",
}

type CMD struct {
	Name   string
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
}

func main() {
	for {
		// Print prompt
		fmt.Fprint(os.Stdout, "\r$ ")
		// Read the user input in raw mode with autocomplete support.
		input := readInputWithAutocomplete(os.Stdin)
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		fields := parseCommand(input)
		if len(fields) == 0 {
			continue
		}

		// Process redirection operators.
		fields, stdoutRedirFile, stderrRedirFile, stdoutAppend, stderrAppend := processRedirectionOperators(fields)
		if len(fields) == 0 {
			continue
		}

		// Map of built-in commands.
		builtins := map[string]func([]string){
			"exit": exitHandler,
			"echo": echoHandler,
			"type": typeHandler,
			"pwd":  pwdHandler,
			"cd":   cdHandler,
		}

		// Execute built-in or external commands.
		if handler, exists := builtins[fields[0]]; exists {
			executeBuiltinWithRedirection(handler, fields, stdoutRedirFile, stderrRedirFile, stdoutAppend, stderrAppend)
		} else {
			executeExternalWithRedirection(fields, stdoutRedirFile, stderrRedirFile, stdoutAppend, stderrAppend)
		}
	}
}

func readInputWithAutocomplete(rd *os.File) string {
	oldState, err := term.MakeRaw(int(rd.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(rd.Fd()), oldState)

	var input string
	r := bufio.NewReader(rd)
	for {
		rn, _, err := r.ReadRune()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			continue
		}
		switch rn {
		case '\x03': // Ctrl+C
			os.Exit(0)
		case '\r', '\n': // Enter key
			fmt.Fprint(os.Stdout, "\r\n")
			return input
		case '\x7F': // Backspace
			if len(input) > 0 {
				input = input[:len(input)-1]
			}
			// Clear line and reprint prompt with input.
			fmt.Fprint(os.Stdout, "\r\x1b[K$ "+input)
		case '\t': // Tab key: autocomplete
			suffix := autocomplete(input)
			if suffix != "" {
				input += suffix + " "
			}
			// Clear line and reprint prompt with updated input.
			fmt.Fprint(os.Stdout, "\r\x1b[K$ "+input)
		default:
			input += string(rn)
			// Clear line and reprint prompt with updated input.
			fmt.Fprint(os.Stdout, "\r\x1b[K$ "+input)
		}
	}
}

// autocomplete returns the missing suffix if the current input uniquely matches a built-in command.
func autocomplete(prefix string) string {
	if prefix == "" {
		return ""
	}
	var matches []string
	for _, cmd := range builtinCMDs {
		// Only autocomplete the command if the entire input is the command (no spaces yet)
		if !strings.Contains(prefix, " ") {
			if strings.HasPrefix(cmd, prefix) && cmd != prefix {
				matches = append(matches, cmd)
			}
		}
	}
	if len(matches) == 1 {
		// Return the missing part of the command.
		return strings.TrimPrefix(matches[0], prefix)
	}
	return ""
}

func processRedirectionOperators(fields []string) ([]string, string, string, bool, bool) {
	stdoutRedirFile := ""
	stderrRedirFile := ""
	stdoutAppend := false
	stderrAppend := false

	var finalFields []string
	i := 0
	for i < len(fields) {
		token := fields[i]
		if token == ">>" || token == "1>>" {
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stdoutRedirFile = fields[i+1]
			stdoutAppend = true
			i += 2
			continue
		}
		if token == ">" || token == "1>" {
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stdoutRedirFile = fields[i+1]
			i += 2
			continue
		}
		if token == "2>" {
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stderrRedirFile = fields[i+1]
			i += 2
			continue
		}
		if token == "2>>" {
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stderrRedirFile = fields[i+1]
			stderrAppend = true
			i += 2
			continue
		}
		finalFields = append(finalFields, token)
		i++
	}

	return finalFields, stdoutRedirFile, stderrRedirFile, stdoutAppend, stderrAppend
}

func executeBuiltinWithRedirection(handler func([]string), args []string,
	stdoutFile, stderrFile string, stdoutAppend, stderrAppend bool) {

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	var stdoutFileHandle *os.File
	if stdoutFile != "" {
		var err error
		if stdoutAppend {
			stdoutFileHandle, err = os.OpenFile(stdoutFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			stdoutFileHandle, err = os.Create(stdoutFile)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stdout redirection:", err)
			return
		}
		os.Stdout = stdoutFileHandle
		defer func() {
			os.Stdout = oldStdout
			stdoutFileHandle.Close()
		}()
	}

	var stderrFileHandle *os.File
	if stderrFile != "" {
		var err error
		if stderrAppend {
			stderrFileHandle, err = os.OpenFile(stderrFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			stderrFileHandle, err = os.Create(stderrFile)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stderr redirection:", err)
			return
		}
		os.Stderr = stderrFileHandle
		defer func() {
			os.Stderr = oldStderr
			stderrFileHandle.Close()
		}()
	}

	handler(args)
}

func executeExternalWithRedirection(fields []string, stdoutFile, stderrFile string,
	stdoutAppend, stderrAppend bool) {

	if stdoutFile == "" && stderrFile == "" {
		executeCommand(fields)
		return
	}

	path, err := exec.LookPath(fields[0])
	if err != nil {
		if stderrFile != "" {
			var f *os.File
			var fileErr error
			if stderrAppend {
				f, fileErr = os.OpenFile(stderrFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			} else {
				f, fileErr = os.Create(stderrFile)
			}
			if fileErr != nil {
				fmt.Fprintln(os.Stderr, "Error opening file for stderr redirection:", fileErr)
				return
			}
			fmt.Fprintln(f, fields[0]+": command not found")
			f.Close()
		} else {
			fmt.Fprintln(os.Stderr, fields[0]+": command not found")
		}
		return
	}

	cmd := exec.Command(path, fields[1:]...)
	if stdoutFile != "" {
		var f *os.File
		var err error
		if stdoutAppend {
			f, err = os.OpenFile(stdoutFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			f, err = os.Create(stdoutFile)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stdout redirection:", err)
			return
		}
		cmd.Stdout = f
		defer f.Close()
	} else {
		cmd.Stdout = os.Stdout
	}

	if stderrFile != "" {
		var f *os.File
		var err error
		if stderrAppend {
			f, err = os.OpenFile(stderrFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			f, err = os.Create(stderrFile)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stderr redirection:", err)
			return
		}
		cmd.Stderr = f
		defer f.Close()
	} else {
		cmd.Stderr = os.Stderr
	}

	cmd.Run()
}

func executeCommand(fields []string) {
	cmd := exec.Command(fields[0], fields[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(fields[0] + ": command not found")
	}
}

func parseCommand(command string) []string {
	var result []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i := 0; i < len(command); i++ {
		c := command[i]
		if escaped {
			if inDoubleQuote {
				if c != '$' && c != '`' && c != '"' && c != '\\' && c != '\n' {
					current.WriteByte('\\')
				}
			}
			current.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' {
			if inSingleQuote {
				current.WriteByte(c)
			} else {
				escaped = true
			}
			continue
		}

		if c == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}

		if c == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if c == ' ' && !inSingleQuote && !inDoubleQuote {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(c)
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

func exitHandler(args []string) {
	os.Exit(0)
}

func echoHandler(args []string) {
	fmt.Println(strings.Join(args[1:], " "))
}

func typeHandler(args []string) {
	if len(args) < 2 {
		fmt.Println("type: missing argument")
		return
	}
	cmd := args[1]
	builtins := map[string]bool{
		"echo": true,
		"exit": true,
		"type": true,
		"pwd":  true,
		"cd":   true,
	}
	if builtins[cmd] {
		fmt.Println(cmd + " is a shell builtin")
	} else if path, err := exec.LookPath(cmd); err == nil {
		fmt.Println(cmd + " is " + path)
	} else {
		fmt.Println(cmd + ": not found")
	}
}

func pwdHandler(args []string) {
	cwd, _ := os.Getwd()
	fmt.Println(cwd)
}

func cdHandler(args []string) {
	if len(args) < 2 {
		fmt.Println("cd: missing argument")
		return
	}
	dir := args[1]
	if dir == "~" {
		dir = os.Getenv("HOME")
	} else if strings.HasPrefix(dir, "~/") {
		dir = os.Getenv("HOME") + dir[1:]
	}
	if err := os.Chdir(dir); err != nil {
		fmt.Printf("cd: %s: No such file or directory\n", dir)
	}
}
