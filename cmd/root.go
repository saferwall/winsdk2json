// Copyright 2022 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "winsdk2json",
	Short: "parse the Windows Win32 API's SDK",
	Long: `WinSdk2JSON - a tool to parse the Windows Win32 SDK into JSON format.
For more details see the github repo at https://github.com/saferwall/winsdk2json`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Vesion number",
	Long:  "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("You are using version 0.0.2")
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(parseCmd)
}
