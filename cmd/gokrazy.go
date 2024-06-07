//go:build gokrazy

package cmd

func init() {
	viper.AddConfigPath("/perm") // path to look for the config file in
}
