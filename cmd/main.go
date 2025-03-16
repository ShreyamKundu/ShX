package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
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
		fmt.Fprint(os.Stdout, "\033[1;32mShX\033[0m ➜ ")
		command, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}

		command = strings.TrimSpace(command)
		fields := parseCommand(command)
		if len(fields) == 0 {
			continue
		}

		// Check if the command is a built-in
		if handler, exists := builtins[fields[0]]; exists {
			handler(fields)
		} else {
			executeCommand(fields)
		}
	}
}

func parseCommand(command string) []string {
	re := regexp.MustCompile(`'[^']*'|"[^"]*"|\S+`)
	matches := re.FindAllString(command, -1)

	var result []string

	for _, match := range matches {
		if (match[0] == '"' && match[len(match)-1] == '"') || (match[0] == '\'' && match[len(match)-1] == '\'') {
			result = append(result, match[1:len(match)-1])
		} else {
			result = append(result, match)
		}
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

	// Handle "~" (home directory)
	if dir == "~" {
		dir = os.Getenv("HOME")
	} else if strings.HasPrefix(dir, "~/") {
		dir = os.Getenv("HOME") + dir[1:]
	}

	if err := os.Chdir(dir); err != nil {
		fmt.Printf("cd: %s: No such file or directory\n", dir)
	}
}
