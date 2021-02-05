// +build gokrazy

package cmd

import "github.com/spf13/viper"

func init() {
	viper.AddConfigPath("/perm") // path to look for the config file in
}
