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
	"strings"

	"github.com/saferwall/winsdk2json/internal/utils"
	"github.com/spf13/cobra"
	"github.com/xlab/c-for-go/translator"
	"modernc.org/cc/v4"
)

// Used for flags.
var (
	sdkapiPath string
	includePath string
)

func init() {

	parseCmd.Flags().StringVarP(&includePath, "include", "i", "./winsdk/10.0.22000.0",
		"Path to the Windows Kits include directory")
	parseCmd.Flags().StringVarP(&sdkapiPath, "sdk-api", "", "./sdk-api",
		"The path to the sdk-api docs directory (https://github.com/MicrosoftDocs/sdk-api)")
}

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Walk through the Windows SDK and parse the Win32 headers",
	Long:  `Walk through the Windows SDK and parse the Win32 headers to produce JSON files.`,
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func run() {

	if _, err := os.Stat(includePath); os.IsNotExist(err) {
		fmt.Print("The include directory does not exist ..")
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

	config.HostSysIncludePaths = config.HostSysIncludePaths[:0]
	config.IncludePaths = config.IncludePaths[:0]
	config.SysIncludePaths = config.SysIncludePaths[:0]

	config.SysIncludePaths = append(config.SysIncludePaths, includePath+"/um")
	config.SysIncludePaths = append(config.SysIncludePaths, includePath+"/shared")
	config.SysIncludePaths = append(config.SysIncludePaths, includePath+"/../14.29.30133/include")
	config.SysIncludePaths = append(config.SysIncludePaths, includePath+"/ucrt")
	config.HostSysIncludePaths = config.SysIncludePaths
	config.IncludePaths = config.SysIncludePaths

	config.Predefined += "\n#define __int64 long long\n"
	config.Predefined += "#define __iamcu__\n"
	config.Predefined += "#define __int32 int\n"
	config.Predefined += "#define NTDDI_WIN7 0x06010000\n"
	config.Predefined += "#define __forceinline __attribute__((always_inline))\n"
	config.Predefined += "#define _AMD64_\n"
	config.Predefined += "#define _M_AMD64\n"
	config.Predefined += "#define __unaligned\n"
	config.Predefined += "#define_MSC_FULL_VER 192930133\n"

	var sources []cc.Source
	sources = append(sources, cc.Source{Name: "<predefined>", Value: config.Predefined})
	sources = append(sources, cc.Source{Name: "<builtin>", Value: cc.Builtin})
	sources = append(sources, cc.Source{Value: code})

	ast, err := cc.Translate(config, sources)
	if err != nil {
		log.Fatalf("cc translate failed with:\n%v", err)
	}

	// d := ast.Scope.Nodes["CreateFileW"][0].(*cc.Declarator)
	// ft := d.Type().(*cc.FunctionType)
	// fmt.Printf("%s\n", ft)
	// for _, v := range ft.Parameters() {
	// 	d = v.Declarator
	// 	t := d.Type()
	// 	attr := t.Attributes()
	// 	if attr != nil {
	// 		attrVal := attr.AttrValue("anno")[0].(cc.StringValue)
	// 		annotation := strings.TrimSpace(string(attrVal))
	// 		fmt.Printf("%s: %s, visibility(%s), (%v)\n", d.Name(), t, attr.Visibility(), annotation)
	// 	}
	// }

	myTranslator, err := translator.New(&translator.Config{})
	if err != nil {
		log.Fatalf("failed to create new translator: %v", err)
	}

	myTranslator.Learn(ast)

	// r := strings.NewReader(ast.TranslationUnit.String())
	// _, err = utils.WriteBytesFile("ast.txt", r)
	// if err != nil {
	// 	log.Fatalf("failed to write ast: %v", err)
	// }

	i := 0
	for _, d := range myTranslator.Declares() {
		if d.Name == "CreateFileW" && !strings.HasPrefix(d.Name, "__builtin_") {
			funcSpec, ok := d.Spec.(*translator.CFunctionSpec)
			if ok {
				retSpec, ok := funcSpec.Return.(*translator.CTypeSpec)
				if ok {

					dllname, err := utils.GetDLLName(d.Position.Filename, d.Name, sdkapiPath)
					if err != nil {
						log.Printf("Failed to get the DLL name for: %s", d.Name)
					}

					returnString := retSpec.Raw
					if retSpec.Raw == "" {
						returnString = retSpec.Base
					}
					fmt.Printf("\n[%s]: %d - %s %s (", dllname, i, returnString, d.Name)
					i++

					funcDecl := ast.Scope.Nodes[d.Name][0].(*cc.Declarator)
					ft := funcDecl.Type().(*cc.FunctionType)
					for idx, param := range funcSpec.Params {
						var annotation string
						paramDecl := ft.Parameters()[idx]
						t := paramDecl.Declarator.Type()
						attr := t.Attributes()
						if attr != nil {
							attrVal := attr.AttrValue("anno")[0].(cc.StringValue)
							annotation = strings.Replace(string(attrVal), "\x00", "", -1)
						}
						paramSpec, ok := param.Spec.(*translator.CTypeSpec)
						if ok {
							fmt.Printf("%s %s %s,", annotation, paramSpec.Raw, param.Name)
						}

					}

					fmt.Printf(")")
				}
			}

		}
	}
	log.Printf("SUCCESS %d", i)
}
