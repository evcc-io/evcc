package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

func main() {
	err := filepath.WalkDir("./i18n", func(filepath string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("%s: %v", filepath, err)
		}

		if d.IsDir() {
			return nil
		}

		var config map[string]any
		if _, err := toml.DecodeFile(filepath, &config); err != nil {
			return fmt.Errorf("%s: %v", filepath, err)
		}

		s := new(strings.Builder)
		enc := toml.NewEncoder(s)
		enc.Indent = ""

		if err := enc.Encode(config); err != nil {
			return fmt.Errorf("%s: %v", filepath, err)
		}

		out := new(strings.Builder)
		sc := bufio.NewScanner(strings.NewReader(s.String()))
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), "[") && strings.Contains(sc.Text(), ".") && out.Len() > 0 {
				fmt.Fprintln(out, "")
			}
			fmt.Fprintln(out, sc.Text())
		}

		if err := os.WriteFile(filepath, []byte(out.String()), 0644); err != nil {
			return fmt.Errorf("%s: %v", filepath, err)
		}

		return nil
	})

	if err != nil {
		panic(err)
	}
}
