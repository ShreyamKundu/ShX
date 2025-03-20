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

		// Look for redirection operator (">" or "1>")
		redirFile := ""
		for i := 0; i < len(fields); i++ {
			if fields[i] == ">" || fields[i] == "1>" {
				if i+1 >= len(fields) {
					fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
					fields = []string{}
					break
				}
				redirFile = fields[i+1]
				// Remove the redirection tokens from the command fields
				fields = append(fields[:i], fields[i+2:]...)
				break
			}
		}

		if len(fields) == 0 {
			continue
		}

		// Check if the command is built-in or external
		if handler, exists := builtins[fields[0]]; exists {
			// For built-ins, temporarily change os.Stdout if redirection is requested.
			if redirFile != "" {
				f, err := os.Create(redirFile)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error opening file for redirection:", err)
					continue
				}
				oldStdout := os.Stdout
				os.Stdout = f
				handler(fields)
				os.Stdout = oldStdout
				f.Close()
			} else {
				handler(fields)
			}
		} else {
			// For external commands, use the file as cmd.Stdout if redirection is requested.
			if redirFile != "" {
				f, err := os.Create(redirFile)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error opening file for redirection:", err)
					continue
				}
				executeCommandWithRedirection(fields, f)
				f.Close()
			} else {
				executeCommand(fields)
			}
		}
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

// Execute external commands
func executeCommand(fields []string) {
	cmd := exec.Command(fields[0], fields[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println(fields[0] + ": command not found")
	}
}

// Execute external commands with redirection for stdout
func executeCommandWithRedirection(fields []string, f *os.File) {
	// Ensure command exists before execution
	path, err := exec.LookPath(fields[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, fields[0]+": command not found")
		return
	}

	cmd := exec.Command(path, fields[1:]...)
	cmd.Stdout = f
	cmd.Stderr = os.Stderr // Ensure errors go to stderr

	if err := cmd.Run(); err != nil {
		// Do NOT print "command not found" on execution errors
		return
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
