package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Define built-in commands using a map
	builtins := map[string]func([]string){
		"exit": exitHandler,
		"echo": echoHandler,
		"type": typeHandler,
		"pwd":  pwdHandler,
		"cd":   cdHandler,
	}

	for {
		fmt.Fprint(os.Stdout, "$ ")
		command, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}

		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}

		fields := parseCommand(command)
		if len(fields) == 0 {
			continue
		}

		// Look for redirection operators
		stdoutRedirFile := ""
		stderrRedirFile := ""
		stdoutAppend := false
		stderrAppend := false

		// Process redirection operators
		fields, stdoutRedirFile, stderrRedirFile, stdoutAppend, stderrAppend = processRedirectionOperators(fields)
		if len(fields) == 0 {
			continue
		}

		// Check if the command is built-in or external
		if handler, exists := builtins[fields[0]]; exists {
			executeBuiltinWithRedirection(handler, fields, stdoutRedirFile, stderrRedirFile, stdoutAppend, stderrAppend)
		} else {
			// For external commands
			executeExternalWithRedirection(fields, stdoutRedirFile, stderrRedirFile, stdoutAppend, stderrAppend)
		}
	}
}

// Process redirection operators in the command
func processRedirectionOperators(fields []string) ([]string, string, string, bool, bool) {
	stdoutRedirFile := ""
	stderrRedirFile := ""
	stdoutAppend := false
	stderrAppend := false

	i := 0
	for i < len(fields) {
		// Check for append redirection operators (>> or 1>>)
		if fields[i] == ">>" || fields[i] == "1>>" {
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stdoutRedirFile = fields[i+1]
			stdoutAppend = true
			// Remove the redirection tokens from the command fields
			fields = append(fields[:i], fields[i+2:]...)
			continue
		}

		// Check for output redirection operators (> or 1>)
		if fields[i] == ">" || fields[i] == "1>" {
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stdoutRedirFile = fields[i+1]
			// Remove the redirection tokens from the command fields
			fields = append(fields[:i], fields[i+2:]...)
			continue
		}

		// Check for stderr redirection operator (2>)
		if fields[i] == "2>" {
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stderrRedirFile = fields[i+1]
			// Remove the redirection tokens from the command fields
			fields = append(fields[:i], fields[i+2:]...)
			continue
		}

		i++
	}

	return fields, stdoutRedirFile, stderrRedirFile, stdoutAppend, stderrAppend
}

// Execute built-in commands with redirection
func executeBuiltinWithRedirection(handler func([]string), args []string,
	stdoutFile, stderrFile string, stdoutAppend, stderrAppend bool) {

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Handle stdout redirection
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

	// Handle stderr redirection
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

	// Execute the command
	handler(args)
}

// Execute external commands with redirection
func executeExternalWithRedirection(fields []string, stdoutFile, stderrFile string,
	stdoutAppend, stderrAppend bool) {

	if stdoutFile == "" && stderrFile == "" {
		executeCommand(fields)
		return
	}

	// Ensure command exists before execution
	path, err := exec.LookPath(fields[0])
	if err != nil {
		if stderrFile != "" {
			// If stderr is being redirected, write the error message to the specified file
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

	// Set up stdout redirection
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

	// Set up stderr redirection
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

// Execute external commands
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

		// Handle escape sequences
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

		// Handle quotes
		if c == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}

		if c == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		// Handle spaces (word boundaries)
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

// Built-in: exit
func exitHandler(args []string) {
	os.Exit(0)
}

// Built-in: echo
func echoHandler(args []string) {
	fmt.Println(strings.Join(args[1:], " "))
}

// Built-in: type
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

// Built-in: pwd
func pwdHandler(args []string) {
	cwd, _ := os.Getwd()
	fmt.Println(cwd)
}

// Built-in: cd
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
