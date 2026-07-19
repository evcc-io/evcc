package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// gendocCmd represents the gendoc command
var gendocCmd = &cobra.Command{
	Use:    "gendoc <output-dir>",
	Short:  "Generate CLI documentation in markdown format",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	Run:    runGendoc,
}

func init() {
	rootCmd.AddCommand(gendocCmd)
}

func runGendoc(cmd *cobra.Command, args []string) {
	outputDir := args[0]

	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.FATAL.Fatalf("Failed to create output directory: %v", err)
	}

	rootCmd.DisableAutoGenTag = true

	// frontmatter title from filename: evcc_password_reset.md -> "evcc password reset"
	filePrepender := func(filename string) string {
		title := strings.ReplaceAll(strings.TrimSuffix(filepath.Base(filename), ".md"), "_", " ")
		return fmt.Sprintf("---\ntitle: \"%s\"\n---\n\n", title)
	}

	// absolute site links without .md extension
	linkHandler := func(name string) string {
		return "/en/reference/cli/" + strings.TrimSuffix(name, ".md")
	}

	if err := doc.GenMarkdownTreeCustom(rootCmd, outputDir, filePrepender, linkHandler); err != nil {
		log.FATAL.Fatalf("Failed to generate documentation: %v", err)
	}

	titleRe := regexp.MustCompile(`(?m)^## evcc.*\n\n`)
	codeRe := regexp.MustCompile(`\n\n\t(.*)\n`)
	blankRe := regexp.MustCompile(`\n{3,}`)

	// make the generated files ready to commit in the docs repo
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		s := string(content)
		// drop the title heading, the frontmatter already contains it
		s = titleRe.ReplaceAllString(s, "")
		// reduce header level by one
		s = strings.ReplaceAll(s, "### ", "## ")
		// lowercase "see also"
		s = strings.ReplaceAll(s, "## SEE ALSO", "## See also")
		// prettier-style list bullets
		s = strings.ReplaceAll(s, "* [", "- [")
		s = strings.ReplaceAll(s, ")\t - ", ") - ")
		// convert single line indented code to backtick surrounded code
		s = codeRe.ReplaceAllString(s, "\n\n```\n$1\n```\n")
		// collapse multiple blank lines and trailing newlines (prettier style)
		s = blankRe.ReplaceAllString(s, "\n\n")
		s = strings.TrimRight(s, "\n") + "\n"

		return os.WriteFile(path, []byte(s), 0o644)
	})
	if err != nil {
		log.FATAL.Fatalf("Failed to modify documentation: %v", err)
	}

	log.INFO.Printf("Documentation generated in %s", outputDir)
}
