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

// supportCmd represents the meter command
var supportCmd = &cobra.Command{
	Use:   "support",
	Short: "Request support at Github Discussions",
	Run:   runsupport,
}

//go:embed support.tpl
var supportTmpl string

func init() {
	rootCmd.AddCommand(supportCmd)
}

func errorString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func runsupport(cmd *cobra.Command, args []string) {
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
	tmpl := template.Must(template.New("support").Funcs(sprig.FuncMap()).Parse(supportTmpl))

	_ = tmpl.Execute(out, map[string]any{
		"CfgFile":    file,
		"CfgError":   errorString(cfgErr),
		"CfgContent": redacted,
		"Version":    server.FormattedVersion(),
	})

	body := out.String()
	uri := "https://github.com/evcc-io/evcc/discussions/new?category=erste-hilfe&body=" + url.QueryEscape(body)

	if err := browser.OpenURL(uri); err != nil {
		log.FATAL.Fatal(err)
	}
}
