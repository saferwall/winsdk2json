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
	includePath string
)

func init() {

	parsev2Cmd.Flags().StringVarP(&includePath, "include", "i", "./mingw-w64/mingw-w64-headers",
		"Path to the mingw-w64 include headers")
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

	if _, err := os.Stat(includePath); os.IsNotExist(err) {
		fmt.Print("The mingw-w64 include directory does not exist ..")
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

	mingwIncludes, err := walkDir(includePath)
	if err != nil {
		log.Fatal(err)
	}

	config.SysIncludePaths = append(config.SysIncludePaths, mingwIncludes...)

	config.Predefined += "\n#define __iamcu__\n"

	var sources []cc.Source
	sources = append(sources, cc.Source{Name: "<predefined>", Value: config.Predefined})
	sources = append(sources, cc.Source{Name: "<builtin>", Value: cc.Builtin})
	//sources = append(sources, cc.Source{Name: "<undefines>", Value: "#undef __cplusplus\n"})
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

	for _, d := range myTranslator.Declares() {
		if d.Name != "EnumMetaFile" {
			funcSpec, ok := d.Spec.(*translator.CFunctionSpec)
			if ok {
				retSpec, ok := funcSpec.Return.(*translator.CTypeSpec)
				if ok {
					returnString := retSpec.Raw
					if retSpec.Raw == "" {
						returnString = retSpec.Base
					}
					fmt.Printf("\n%s %s (", returnString, d.Name)
					for _, param := range funcSpec.Params {
						paramSpec, ok := param.Spec.(*translator.CTypeSpec)
						if ok {
							fmt.Printf("%s %s,", paramSpec.Raw, param.Name)
						}
					}
				}
			}
			fmt.Printf(")")

		}
	}
	log.Print("SUCCESS")
}
