package cmd

import (
	"github.com/evcc-io/evcc/api"
	"github.com/spf13/cobra"
)

// handleCurtailFlag handles the --curtail flag for a given device.
// Returns true if the flag was used.
func handleCurtailFlag(cmd *cobra.Command, v any) bool {
	if !cmd.Flags().Changed(flagCurtail) {
		return false
	}

	val, err := cmd.Flags().GetInt(flagCurtail)
	if err != nil {
		log.ERROR.Println("curtail:", err)
		return true
	}

	if vv, ok := api.Cap[api.Curtailer](v); ok {
		if err := vv.Curtail(val > 0); err != nil {
			log.ERROR.Println("curtail:", err)
		}
	} else {
		log.ERROR.Println("curtail: not implemented")
	}

	return true
}

// handleDimFlag handles the --dim flag for a given device.
// Returns true if the flag was used.
func handleDimFlag(cmd *cobra.Command, v any) bool {
	if !cmd.Flags().Changed(flagDim) {
		return false
	}

	val, err := cmd.Flags().GetInt(flagDim)
	if err != nil {
		log.ERROR.Println("dim:", err)
		return true
	}

	if vv, ok := api.Cap[api.Dimmer](v); ok {
		if err := vv.Dim(val > 0); err != nil {
			log.ERROR.Println("dim:", err)
		}
	} else {
		log.ERROR.Println("dim: not implemented")
	}

	return true
}
