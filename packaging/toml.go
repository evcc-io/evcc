package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

func process(filepath string) error {
	in, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	pre := new(strings.Builder)
	{
		sc := bufio.NewScanner(bytes.NewReader(in))
		for sc.Scan() {
			col := strings.SplitN(sc.Text(), "=", 2)
			if len(col) < 2 {
				fmt.Fprintln(pre, sc.Text())
				continue
			}

			key := strings.TrimSpace(col[0])
			val := strings.TrimSpace(col[1])

			if strings.HasPrefix(val, "\"") {
				fmt.Fprintln(pre, sc.Text())
				continue
			}

			quote := `"`
			if strings.Contains(val, quote) {
				quote = `'`
			}

			fmt.Fprintf(pre, "%s = %s%s%s\n", key, quote, val, quote)
		}
	}

	var config map[string]any
	if _, err := toml.Decode(pre.String(), &config); err != nil {
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

	if err := os.WriteFile(filepath, []byte(out.String()), 0o644); err != nil {
		return fmt.Errorf("%s: %v", filepath, err)
	}

	return nil
}

func main() {
	var err error
	if len(os.Args) > 1 {
		err = process(os.Args[1])
	} else {
		err = filepath.WalkDir("./i18n", func(filepath string, d fs.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("%s: %v", filepath, err)
			}

			if d.IsDir() {
				return nil
			}

			return process(filepath)
		})
	}

	if err != nil {
		panic(err)
	}
}
