// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"flag"
	"log"
	"os"
	"runtime"

	"github.com/saferwall/winsdk2json/internal/utils"
	"github.com/spf13/cobra"
	"modernc.org/cc/v4"
)

// Used for flags.
var (
	crtPath string
)

func init() {

	parsev2Cmd.Flags().StringVarP(&crtPath, "sdk", "", "C:\\Program Files (x86)\\Windows Kits\\10\\Include\\10.0.22000.0\\ucrt\\",
		"The path to the windows sdk directory")
}

var parsev2Cmd = &cobra.Command{
	Use:   "ast",
	Short: "Walk through the Windows SDK and parse the Win32 headers",
	Long:  `Walk through the Windows SDK and parse the Win32 headers to produce JSON files.`,
	Run: func(cmd *cobra.Command, args []string) {
		runv2()
	},
}

func runv2() {

	if crtPath == "" {
		flag.Usage()
		os.Exit(0)
	}

	code, err := utils.ReadAll(".\\assets\\v2.inc")
	if err != nil {
		log.Fatal("reading v2 header file failed")
	}

	config, err := cc.NewConfig(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		log.Fatal(err)
	}

	var sources []cc.Source
	sources = append(sources, cc.Source{Name: "<predefined>", Value: config.Predefined})
	sources = append(sources, cc.Source{Name: "<builtin>", Value: cc.Builtin})
	sources = append(sources, cc.Source{Value: code})

	ast, err := cc.Parse(config, sources)
	if err != nil {
		log.Fatalf("parse cpp source file failed with: %v", err)
	}
	//log.Print(ast)
	log.Print(ast.TranslationUnit.String())

}
