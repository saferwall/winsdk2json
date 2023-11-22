// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	version = "v0.3.0"
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
	Short: "Version number",
	Long:  "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("You are using version %s", version)
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(parseCmdOld)
}
