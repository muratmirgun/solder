// Command solder generates solder_gen.go files from injectors.
//
//	solder gen [packages]    write solder_gen.go for each package with injectors
//	solder check [packages]  report errors and out-of-date files, write nothing
//
// Packages default to ".".
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/muratmirgun/solder/internal/codegen"
)

func main() {
	if len(os.Args) < 2 || (os.Args[1] != "gen" && os.Args[1] != "check") {
		fmt.Fprintln(os.Stderr, "usage: solder gen|check [packages]")
		os.Exit(2)
	}
	os.Exit(run(os.Args[1], os.Args[2:]))
}

func run(cmd string, patterns []string) int {
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "solder:", err)
		return 1
	}
	files, genErr := codegen.Generate(wd, patterns...)
	code := 0
	for _, f := range files {
		name, _ := filepath.Rel(wd, f.Path)
		if cmd == "check" {
			if old, _ := os.ReadFile(f.Path); !bytes.Equal(old, f.Content) {
				fmt.Fprintf(os.Stderr, "solder: %s is out of date; run solder gen\n", name)
				code = 1
			}
			continue
		}
		if err := os.WriteFile(f.Path, f.Content, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "solder:", err)
			code = 1
			continue
		}
		fmt.Println("wrote", name)
	}
	if genErr != nil {
		fmt.Fprintln(os.Stderr, genErr)
		code = 1
	}
	return code
}
