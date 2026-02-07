package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func RunMenu(baseDir string) int {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nTools:")
	fmt.Println("  1) File search")
	fmt.Println("  2) Rename files")
	fmt.Println("  0) Cancel")
	fmt.Print("\n> ")

	choice := readLine(reader)
	switch choice {
	case "1":
		return RunFileSearch(baseDir, reader)
	case "2":
		return RunRename(baseDir, reader)
	case "0", "":
		return 0
	default:
		fmt.Println("Invalid choice.")
		return 0
	}
}

func prompt(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	text, _ := r.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}
	return text
}

func readLine(r *bufio.Reader) string {
	s, _ := r.ReadString('\n')
	return strings.TrimSpace(s)
}
