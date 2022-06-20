package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "0.1.1"

var verCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("version: %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(verCmd)
}
