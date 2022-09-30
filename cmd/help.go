//go:build !windows

package cmd

import (
	"bytes"
	_ "embed"
	"net/url"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// helpCmd represents the meter command
var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Request help at Github Discussions",
	Run:   runHelp,
}

//go:embed help.tpl
var helpTmpl string

func init() {
	rootCmd.AddCommand(helpCmd)
}

func errorString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func runHelp(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s", server.FormattedVersion())

	cfgErr := loadConfigFile(&conf)

	file, pathErr := filepath.Abs(cfgFile)
	if pathErr != nil {
		file = cfgFile
	}

	var redacted string
	if src, err := os.ReadFile(cfgFile); err == nil {
		redacted = redact(string(src))
	}

	out := new(bytes.Buffer)
	tmpl := template.Must(template.New("help").Funcs(sprig.FuncMap()).Parse(helpTmpl))
	tmpl.Execute(out, map[string]any{
		"CfgFile":    file,
		"CfgError":   errorString(cfgErr),
		"CfgContent": redacted,
		"Version":    server.FormattedVersion(),
	})

	body := out.String()
	uri := "https://github.com/evcc-io/evcc/discussions/new?category=erste-hilfe&body=" + url.QueryEscape(body)

	browser.OpenURL(uri)
}
