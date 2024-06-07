//go:build gokrazy

package cmd

func init() {
	vpr.AddConfigPath("/perm") // path to look for the config file in
}
