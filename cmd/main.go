package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var _ = fmt.Fprint

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stdout, "$ ")
		// Wait for user input
		command, err := reader.ReadString('\n')

		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}

		command = strings.TrimSpace(command)
		switch {
		case command == "exit 0":
			os.Exit(0)
		case strings.HasPrefix(command, "echo "):
			fmt.Println(strings.TrimPrefix(command, "echo "))
		case strings.HasPrefix(command, "type "):
			checkType(command)
		default:
			fmt.Println(command + ": command not found")
		}
	}
}

func checkType(command string) {
	args := strings.Fields(command)
	if len(args) < 2 {
		fmt.Println("type: missing argument")
		return
	}

	builtins := map[string]bool{
		"echo": true,
		"exit": true,
		"type": true,
	}

	cmd := args[1]
	if builtins[cmd] {
		fmt.Println(cmd + " is a shell builtin")
	} else if path, err := exec.LookPath(cmd); err == nil {
		fmt.Println(cmd + " is " + path)
	} else {
		fmt.Println(cmd + ": not found")
	}
}
