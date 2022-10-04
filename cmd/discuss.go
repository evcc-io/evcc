package cmd

import (
	"bytes"
	_ "embed"
	"net/url"
	"os"
	"path/filepath"
	"text/template"

	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// discussCmd represents the discuss command
var discussCmd = &cobra.Command{
	Use:   "discuss",
	Short: "Request support at Github Discussions (https://github.com/evcc-io/evcc/discussions/categories/erste-hilfe)",
	Run:   runDiscuss,
}

//go:embed discuss.tpl
var discussTmpl string

func init() {
	rootCmd.AddCommand(discussCmd)
}

func errorString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func runDiscuss(cmd *cobra.Command, args []string) {
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

	tmpl := template.Must(template.New("discuss").Parse(discussTmpl))

	out := new(bytes.Buffer)
	_ = tmpl.Execute(out, map[string]any{
		"CfgFile":    file,
		"CfgError":   errorString(cfgErr),
		"CfgContent": redacted,
		"Version":    server.FormattedVersion(),
	})

	body := out.String()
	uri := "https://github.com/evcc-io/evcc/discussions/new?category=erste-hilfe&body=" + url.QueryEscape(body)

	if err := browser.OpenURL(uri); err != nil {
		log.FATAL.Println("Could not open browser.")
		log.FATAL.Println("Go to https://github.com/evcc-io/evcc/discussions/new?category=erste-hilfe and post the following:")
		log.FATAL.Println(body)
	}
}
