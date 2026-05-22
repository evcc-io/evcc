package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/charger/easee"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var easeeOcppCmd = &cobra.Command{
	Use:   "easee-ocpp",
	Short: "Manage Easee local OCPP configuration",
}

var easeeOcppEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable local OCPP on Easee charger",
	Run:   runEaseeOcppEnable,
}

var easeeOcppDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable local OCPP on Easee charger",
	Run:   runEaseeOcppDisable,
}

var easeeOcppStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Easee local OCPP configuration",
	Run:   runEaseeOcppStatus,
}

func init() {
	rootCmd.AddCommand(easeeOcppCmd)
	easeeOcppCmd.AddCommand(easeeOcppEnableCmd)
	easeeOcppCmd.AddCommand(easeeOcppDisableCmd)
	easeeOcppCmd.AddCommand(easeeOcppStatusCmd)

	easeeOcppCmd.PersistentFlags().String("user", "", "Easee account email")
	easeeOcppCmd.PersistentFlags().String("password", "", "Easee account password")
	easeeOcppCmd.PersistentFlags().String("charger", "", "Easee charger serial number")

	_ = easeeOcppCmd.MarkPersistentFlagRequired("user")
	_ = easeeOcppCmd.MarkPersistentFlagRequired("password")
	_ = easeeOcppCmd.MarkPersistentFlagRequired("charger")

	easeeOcppEnableCmd.Flags().String("url", "", "OCPP Central System URL (e.g. ws://192.168.1.100:8887/)")
	_ = easeeOcppEnableCmd.MarkFlagRequired("url")
}

type localOcppConnectionDetails struct {
	ConnectivityMode        string                  `json:"connectivityMode"`
	WebsocketConnectionArgs *localOcppWebsocketArgs `json:"websocketConnectionArgs,omitempty"`
}

type localOcppWebsocketArgs struct {
	URL string `json:"url"`
}

type localOcppVersionResponse struct {
	Version string `json:"version"`
}

type localOcppApplyRequest struct {
	Version string `json:"version"`
}

type localOcppStatusResponse struct {
	ConnectivityMode        string                  `json:"connectivityMode"`
	ChargePointID           string                  `json:"chargePointId"`
	WebsocketConnectionArgs *localOcppWebsocketArgs `json:"websocketConnectionArgs"`
}

func easeeOcppClient(user, password string) (*request.Helper, error) {
	log := util.NewLogger("easee-ocpp")
	log.Redact(user, password)

	ts, err := easee.TokenSource(log, user, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	client := request.NewHelper(log)
	client.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   client.Client.Transport,
	}

	return client, nil
}

func easeeOcppApplyConfig(client *request.Helper, serial string, details localOcppConnectionDetails) error {
	uri := fmt.Sprintf("%s/v1/connection-details/%s", easee.LocalOcppAPI, serial)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(details), request.JSONEncoding)
	if err != nil {
		return err
	}

	var version localOcppVersionResponse
	if err := client.DoJSON(req, &version); err != nil {
		return fmt.Errorf("storing config: %w", err)
	}

	apply := localOcppApplyRequest{Version: version.Version}
	uri = fmt.Sprintf("%s/v1/connections/chargers/%s", easee.LocalOcppAPI, serial)
	req, err = request.New(http.MethodPost, uri, request.MarshalJSON(apply), request.JSONEncoding)
	if err != nil {
		return err
	}

	if _, err := client.DoBody(req); err != nil {
		return fmt.Errorf("applying config: %w", err)
	}

	return nil
}

func runEaseeOcppEnable(cmd *cobra.Command, args []string) {
	user, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	charger, _ := cmd.Flags().GetString("charger")

	client, err := easeeOcppClient(user, password)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	ocppURL, _ := cmd.Flags().GetString("url")

	details := localOcppConnectionDetails{
		ConnectivityMode:        "DualProtocol",
		WebsocketConnectionArgs: &localOcppWebsocketArgs{URL: ocppURL},
	}

	if err := easeeOcppApplyConfig(client, charger, details); err != nil {
		log.FATAL.Fatal(err)
	}

	fmt.Printf("Local OCPP enabled for charger %s\n", charger)
	fmt.Printf("OCPP Central System URL: %s\n", ocppURL)
}

func runEaseeOcppDisable(cmd *cobra.Command, args []string) {
	user, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	charger, _ := cmd.Flags().GetString("charger")

	client, err := easeeOcppClient(user, password)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	// GET current config, strip charger-appended serial from URL, set OcppOff
	uri := fmt.Sprintf("%s/v1/connection-details/%s", easee.LocalOcppAPI, charger)

	var current localOcppConnectionDetails
	if err := client.GetJSON(uri, &current); err != nil {
		log.FATAL.Fatal(err)
	}

	current.ConnectivityMode = "OcppOff"
	if current.WebsocketConnectionArgs != nil {
		current.WebsocketConnectionArgs.URL = strings.TrimSuffix(current.WebsocketConnectionArgs.URL, charger)
	}

	if err := easeeOcppApplyConfig(client, charger, current); err != nil {
		log.FATAL.Fatal(err)
	}

	fmt.Printf("Local OCPP disabled for charger %s\n", charger)
}

func runEaseeOcppStatus(cmd *cobra.Command, args []string) {
	user, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	charger, _ := cmd.Flags().GetString("charger")

	client, err := easeeOcppClient(user, password)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	uri := fmt.Sprintf("%s/v1/connection-details/%s", easee.LocalOcppAPI, charger)

	var status localOcppStatusResponse
	if err := client.GetJSON(uri, &status); err != nil {
		log.FATAL.Fatal(err)
	}

	fmt.Printf("Charger:           %s\n", charger)
	fmt.Printf("Connectivity mode: %s\n", status.ConnectivityMode)
	fmt.Printf("Charge point ID:   %s\n", status.ChargePointID)
	if status.WebsocketConnectionArgs != nil {
		fmt.Printf("OCPP URL:          %s\n", status.WebsocketConnectionArgs.URL)
	}
}
