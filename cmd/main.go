package main

import (
	"bufio"
	"fmt"
	"os"
)

var _ = fmt.Fprint

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprint(os.Stdout, "$ ")
	// Wait for user input
	_, err := reader.ReadString('\n')

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		os.Exit(1)
	}
}
