package cmd

import "github.com/spf13/cobra"

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "ezjob",
	Short: "Easy Job",
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "ezjob config file")
}

func Execute() error {
	return rootCmd.Execute()
}
