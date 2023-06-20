// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/saferwall/winsdk2json/internal/utils"
	"github.com/spf13/cobra"
	"github.com/xlab/c-for-go/translator"
	"modernc.org/cc/v4"
)

// Used for flags.
var (
	sdkPath  string
	msvcPath string
)

func init() {

	parsev2Cmd.Flags().StringVarP(&sdkPath, "sdk", "s", "C:\\Program Files (x86)\\Windows Kits\\10\\Include\\10.0.22000.0\\",
		"The path to the windows sdk directory")

	parsev2Cmd.Flags().StringVarP(&msvcPath, "msvc", "m", "C:\\Program Files (x86)\\Microsoft Visual Studio\\2019\\Community\\VC\\Tools\\MSVC\\14.29.30133",
		"The path to the MSVC directory")
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

	if _, err := os.Stat(sdkPath); os.IsNotExist(err) {
		fmt.Print("The Windows SDK directory does not exist ..")
		flag.Usage()
		os.Exit(0)
	}
	if _, err := os.Stat(msvcPath); os.IsNotExist(err) {
		fmt.Print("The Windows MSVC directory does not exist ..")
		flag.Usage()
		os.Exit(0)
	}

	filePath := filepath.Join("assets", "header.h")
	code, err := utils.ReadAll(filePath)
	if err != nil {
		log.Fatalf("reading header.h failed: %v", err)
	}

	config, err := cc.NewConfig(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		log.Fatal(err)
	}

	sdkIncludePaths, err := walkDir(sdkPath)
	if err != nil {
		log.Fatal(err)
	}
	msvcIncludePaths, err := walkDir(msvcPath)
	if err != nil {
		log.Fatal(err)
	}

	config.HostSysIncludePaths = config.HostSysIncludePaths[:0]
	config.IncludePaths = config.IncludePaths[:0]
	config.SysIncludePaths = config.SysIncludePaths[:0]

	config.SysIncludePaths = append(sdkIncludePaths, msvcIncludePaths...)

	config.HostSysIncludePaths = config.SysIncludePaths
	config.IncludePaths = config.SysIncludePaths

	var sources []cc.Source
	//sources = append(sources, cc.Source{Name: "<builtin>", Value: "typedef unsigned size_t; int __predefined_declarator;"})
	sources = append(sources, cc.Source{Name: "<predefined>", Value: config.Predefined})
	sources = append(sources, cc.Source{Name: "<undefines>", Value: "#undef __cplusplus\n#undef _WIN64\n"})
	//sources = append(sources, cc.Source{Name: "<builtin>", Value: cc.Builtin})
	sources = append(sources, cc.Source{Value: code})

	// ast, err := cc.Parse(config, sources)
	ast, err := cc.Translate(config, sources)
	if err != nil {
		log.Fatalf("parse cpp source file failed with:\n%v", err)
	}

	myTranslator, err := translator.New(&translator.Config{})
	if err != nil {
		log.Fatalf("failed to create new translator: %v", err)
	}

	myTranslator.Learn(ast)

	//r := strings.NewReader(ast.TranslationUnit.String())
	//utils.WriteBytesFile("ast.txt", r)
	//log.Print(ast)
	//log.Print(ast.TranslationUnit.String())
}
