package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
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
		printPrompt()
		input := readInputWithAutocomplete(os.Stdin)
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		fields := parseCommand(input)
		if len(fields) == 0 {
			continue
		}

		fields, stdoutFile, stderrFile, stdoutAppend, stderrAppend := processRedirectionOperators(fields)
		if len(fields) == 0 {
			continue
		}

		builtins := map[string]func([]string){
			"exit": exitHandler,
			"echo": echoHandler,
			"type": typeHandler,
			"pwd":  pwdHandler,
			"cd":   cdHandler,
		}

		if handler, exists := builtins[fields[0]]; exists {
			executeBuiltinWithRedirection(handler, fields, stdoutFile, stderrFile, stdoutAppend, stderrAppend)
		} else {
			executeExternalWithRedirection(fields, stdoutFile, stderrFile, stdoutAppend, stderrAppend)
		}
	}
}

func printPrompt() {
	fmt.Fprint(os.Stdout, "\r$ ")
}

// Track the last prefix and tab count for double-tab behavior
var lastTabPrefix string
var tabPressCount int

func readInputWithAutocomplete(rd *os.File) string {
	oldState, err := term.MakeRaw(int(rd.Fd()))
	if err != nil {
		panic(err)
	}
	// The terminal state is restored explicitly on Ctrl+C or when input is done.
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
			term.Restore(int(rd.Fd()), oldState)
			os.Exit(0)
		case '\r', '\n': // Enter key
			fmt.Fprint(os.Stdout, "\r\n")
			// Reset tab count on Enter
			tabPressCount = 0
			lastTabPrefix = ""
			return input
		case '\x7F': // Backspace
			if len(input) > 0 {
				input = input[:len(input)-1]
			}
			// Reset tab count on any modification
			tabPressCount = 0
			lastTabPrefix = ""
			printPromptWithInput(input)
		case '\t': // Tab key: autocomplete
			prefix := input
			// If there's a space, we're not autocompleting a command
			if strings.Contains(prefix, " ") {
				tabPressCount = 0
				lastTabPrefix = ""
				break
			}

			// If prefix is different from last tab, reset counter
			if prefix != lastTabPrefix {
				tabPressCount = 0
				lastTabPrefix = prefix
			}

			tabPressCount++
			result, matches := autocomplete(prefix, tabPressCount)

			if len(matches) > 1 && tabPressCount > 1 {
				// On second tab press with multiple matches, show all possibilities
				printAllMatches(matches)
				printPromptWithInput(input)
			} else if result != "" {
				input += result + " "
				printPromptWithInput(input)
				// Reset tab count after successful completion
				tabPressCount = 0
				lastTabPrefix = ""
			} else {
				// Ring the bell on first tab or no matches
				fmt.Fprint(os.Stdout, "\a")
			}
		default:
			input += string(rn)
			// Reset tab count on any modification
			tabPressCount = 0
			lastTabPrefix = ""
			printPromptWithInput(input)
		}
	}
}

func printAllMatches(matches []string) {
	fmt.Fprint(os.Stdout, "\r\n")
	fmt.Fprint(os.Stdout, strings.Join(matches, "  "))
	fmt.Fprint(os.Stdout, "\r\n")
}

func printPromptWithInput(input string) {
	// Clear line and reprint prompt with updated input.
	fmt.Fprint(os.Stdout, "\r\x1b[K$ "+input)
}

func autocomplete(prefix string, tabCount int) (string, []string) {
	if prefix == "" {
		return "", nil
	}

	var matches []string
	// First, check built-in commands.
	for _, cmd := range builtinCMDs {
		if strings.HasPrefix(cmd, prefix) && cmd != prefix {
			matches = append(matches, cmd)
		}
	}

	// If no built-in command matches, search for external executables in PATH.
	if len(matches) == 0 {
		pathEnv := os.Getenv("PATH")
		dirs := strings.Split(pathEnv, ":")
		found := make(map[string]bool)
		for _, dir := range dirs {
			files, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, file := range files {
				// Skip directories.
				if file.IsDir() {
					continue
				}
				name := file.Name()
				// Check if the file name starts with the prefix and is not exactly the prefix.
				if strings.HasPrefix(name, prefix) && name != prefix {
					if !found[name] {
						found[name] = true
						matches = append(matches, name)
					}
				}
			}
		}
	}

	if len(matches) == 0 {
		return "", nil
	}

	// Sort matches for consistent display
	sort.Strings(matches)

	if len(matches) == 1 {
		return strings.TrimPrefix(matches[0], prefix), nil
	}

	return "", matches
}

func processRedirectionOperators(fields []string) ([]string, string, string, bool, bool) {
	stdoutFile := ""
	stderrFile := ""
	stdoutAppend := false
	stderrAppend := false
	var finalFields []string

	for i := 0; i < len(fields); {
		token := fields[i]
		switch token {
		case ">>", "1>>":
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stdoutFile = fields[i+1]
			stdoutAppend = true
			i += 2
		case ">", "1>":
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stdoutFile = fields[i+1]
			i += 2
		case "2>":
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stderrFile = fields[i+1]
			i += 2
		case "2>>":
			if i+1 >= len(fields) {
				fmt.Fprintln(os.Stderr, "syntax error: no file specified for redirection")
				return []string{}, "", "", false, false
			}
			stderrFile = fields[i+1]
			stderrAppend = true
			i += 2
		default:
			finalFields = append(finalFields, token)
			i++
		}
	}
	return finalFields, stdoutFile, stderrFile, stdoutAppend, stderrAppend
}

func executeBuiltinWithRedirection(
	handler func([]string),
	args []string,
	stdoutFile, stderrFile string,
	stdoutAppend, stderrAppend bool,
) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	if stdoutFile != "" {
		file, err := openFile(stdoutFile, stdoutAppend)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stdout redirection:", err)
			return
		}
		os.Stdout = file
		defer func() {
			os.Stdout = oldStdout
			file.Close()
		}()
	}

	if stderrFile != "" {
		file, err := openFile(stderrFile, stderrAppend)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stderr redirection:", err)
			return
		}
		os.Stderr = file
		defer func() {
			os.Stderr = oldStderr
			file.Close()
		}()
	}

	handler(args)
}

func executeExternalWithRedirection(
	fields []string,
	stdoutFile, stderrFile string,
	stdoutAppend, stderrAppend bool,
) {
	if stdoutFile == "" && stderrFile == "" {
		executeCommand(fields)
		return
	}

	path, err := exec.LookPath(fields[0])
	if err != nil {
		outputError(fields[0], stderrFile, stderrAppend)
		return
	}

	cmd := exec.Command(path, fields[1:]...)

	if stdoutFile != "" {
		if file, err := openFile(stdoutFile, stdoutAppend); err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stdout redirection:", err)
			return
		} else {
			cmd.Stdout = file
			defer file.Close()
		}
	} else {
		cmd.Stdout = os.Stdout
	}

	if stderrFile != "" {
		if file, err := openFile(stderrFile, stderrAppend); err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stderr redirection:", err)
			return
		} else {
			cmd.Stderr = file
			defer file.Close()
		}
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

func openFile(fileName string, appendMode bool) (*os.File, error) {
	if appendMode {
		return os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
	return os.Create(fileName)
}

func outputError(cmdName, stderrFile string, appendMode bool) {
	message := cmdName + ": command not found"
	if stderrFile != "" {
		if file, err := openFile(stderrFile, appendMode); err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file for stderr redirection:", err)
		} else {
			fmt.Fprintln(file, message)
			file.Close()
		}
	} else {
		fmt.Fprintln(os.Stderr, message)
	}
}

func parseCommand(command string) []string {
	var result []string
	var current strings.Builder
	inSingleQuote, inDoubleQuote, escaped := false, false, false

	for i := 0; i < len(command); i++ {
		c := command[i]

		if escaped {
			if inDoubleQuote && c != '$' && c != '`' && c != '"' && c != '\\' && c != '\n' {
				current.WriteByte('\\')
			}
			current.WriteByte(c)
			escaped = false
			continue
		}

		switch c {
		case '\\':
			if inSingleQuote {
				current.WriteByte(c)
			} else {
				escaped = true
			}
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			} else {
				current.WriteByte(c)
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			} else {
				current.WriteByte(c)
			}
		case ' ':
			if !inSingleQuote && !inDoubleQuote {
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(c)
			}
		default:
			current.WriteByte(c)
		}
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
	switch {
	case dir == "~":
		dir = os.Getenv("HOME")
	case strings.HasPrefix(dir, "~/"):
		dir = os.Getenv("HOME") + dir[1:]
	}

	if err := os.Chdir(dir); err != nil {
		fmt.Printf("cd: %s: No such file or directory\n", dir)
	}
}
