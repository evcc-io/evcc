// Command eudataact exercises the live EU Data Act login and data flow.
//
// Run from the repository root, for example:
//
//	EUDA_PASSWORD='secret' go run ./vehicle/vw/eudataact/tools -brand Volkswagen user@example.com
//
// If the password is not supplied through EUDA_PASSWORD, the command prompts
// without echoing it.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/vw/eudataact"
	"golang.org/x/term"
)

const (
	emailEnv    = "EUDA_EMAIL"
	passwordEnv = "EUDA_PASSWORD"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, in *os.File, out, errOut *os.File) int {
	fs := flag.NewFlagSet("eudataact", flag.ContinueOnError)
	fs.SetOutput(errOut)
	brand := fs.String("brand", "Volkswagen", "VW group brand")
	vin := fs.String("vin", "", "vehicle VIN (defaults to the first linked vehicle)")
	email := fs.String("email", os.Getenv(emailEnv), "account e-mail, or use "+emailEnv)
	password := os.Getenv(passwordEnv)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(errOut, "error: expected at most one positional e-mail argument")
		return 2
	}
	if fs.NArg() == 1 {
		*email = fs.Arg(0)
	}
	if *email == "" {
		fmt.Fprint(errOut, "E-mail: ")
		if _, err := fmt.Fscanln(in, email); err != nil {
			fmt.Fprintf(errOut, "error reading e-mail: %v\n", err)
			return 2
		}
	}
	if password == "" {
		var err error
		password, err = readPassword(in, errOut)
		if err != nil {
			fmt.Fprintf(errOut, "error reading password: %v\n", err)
			return 2
		}
	}
	if *email == "" || password == "" {
		fmt.Fprintln(errOut, "error: e-mail and password are required")
		return 2
	}

	util.LogLevel("trace", nil)
	log := util.NewLogger("eudataact").Redact(*email, password)
	fmt.Fprintf(out, "Logging in for %s...\n", *brand)
	api, err := eudataact.NewAPI(log, *brand, *email, password)
	if err != nil {
		fmt.Fprintf(errOut, "login failed: %s\n", redact(err.Error(), *email, password))
		return 1
	}
	fmt.Fprintln(out, "Login successful.")

	vehicles, err := api.Vehicles()
	if err != nil {
		fmt.Fprintf(errOut, "listing vehicles failed: %s\n", redact(err.Error(), *email, password))
		return 1
	}
	if len(vehicles) == 0 {
		fmt.Fprintln(errOut, "no linked vehicles returned")
		return 1
	}
	for _, vehicle := range vehicles {
		fmt.Fprintf(out, "  %s  %s\n", vehicle.Vin(), vehicle.Name())
	}

	selected := vehicles[0]
	if *vin != "" {
		found := false
		for _, vehicle := range vehicles {
			if strings.EqualFold(vehicle.Vin(), *vin) {
				selected = vehicle
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(errOut, "vehicle %q was not returned by the portal\n", *vin)
			return 1
		}
	}

	fmt.Fprintf(out, "Downloading newest dataset for %s...\n", selected.Vin())
	dataset, err := api.LatestDataset(selected.Vin())
	if err != nil {
		fmt.Fprintf(errOut, "dataset check failed: %s\n", redact(err.Error(), *email, password))
		return 1
	}
	fmt.Fprintf(out, "  %s  created=%s  points=%d\n",
		dataset.Name, dataset.CreatedOn.Format("2006-01-02 15:04:05 MST"), dataset.PointCount)
	return 0
}

func redact(value string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[REDACTED]")
		}
	}
	return value
}

func readPassword(in *os.File, out *os.File) (string, error) {
	if !term.IsTerminal(int(in.Fd())) {
		return "", errors.New("set EUDA_PASSWORD when standard input is not a terminal")
	}
	fmt.Fprint(out, "Password: ")
	value, err := term.ReadPassword(int(in.Fd()))
	fmt.Fprintln(out)
	return string(value), err
}
