// +build eebus

package cmd

import (
	"bufio"
	"crypto/x509/pkix"
	"fmt"
	"strings"

	certhelper "github.com/amp-x/eebus/cert"
	"github.com/amp-x/eebus/communication"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// teslaCmd represents the vehicle command
var eebusCmd = &cobra.Command{
	Use:   "eebus-cert",
	Short: "Generate EEBUS certificate for using EEBUS compatible chargers",
	Run:   runEEBUSCert,
}

func init() {
	rootCmd.AddCommand(eebusCmd)
}

func generateEEBUSCert() {
	details := communication.ManufacturerDetails{
		DeviceName:    "EVCC",
		DeviceCode:    "EVCC_HEMS_01",
		DeviceAddress: "EVCC_HEMS",
		BrandName:     "EVCC",
	}

	subject := pkix.Name{
		CommonName:   details.DeviceCode,
		Country:      []string{"DE"},
		Organization: []string{details.BrandName},
	}

	cert, err := certhelper.CreateCertificate(true, subject)
	if err != nil {
		fmt.Println("Could not create certificate")
		return
	}

	certValue, keyValue, err := certhelper.GetX509KeyPair(cert)
	if err != nil {
		fmt.Println("Could not process generated certificate")
		return
	}

	fmt.Println()
	fmt.Println("Add the following configuration to the config:")
	fmt.Println()
	fmt.Println("eebus:")
	fmt.Println("  certificate:")
	fmt.Println("    public: |")
	scanner := bufio.NewScanner(strings.NewReader(certValue))
	for scanner.Scan() {
		fmt.Println("      ", scanner.Text())
	}
	fmt.Println("    private: |")
	scanner = bufio.NewScanner(strings.NewReader(keyValue))
	for scanner.Scan() {
		fmt.Println("      ", scanner.Text())
	}
}

func runEEBUSCert(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	generateEEBUSCert()
}
