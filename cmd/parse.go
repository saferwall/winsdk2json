// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/saferwall/winsdk2json/internal/entity"
	log "github.com/saferwall/winsdk2json/internal/logger"
	"github.com/saferwall/winsdk2json/internal/utils"
	"github.com/spf13/cobra"
	"github.com/xlab/c-for-go/translator"
	"modernc.org/cc/v4"
)

// Used for flags.
var (
	sdkapiPath   string
	includePath  string
	dumpAST      bool
	genJSONForUI bool
)

func init() {

	parseCmd.Flags().StringVarP(&includePath, "include", "i", "./winsdk/10.0.22000.0",
		"Path to the Windows Kits include directory")
	parseCmd.Flags().StringVarP(&sdkapiPath, "sdk-api", "", "./sdk-api",
		"The path to the sdk-api docs directory (https://github.com/MicrosoftDocs/sdk-api)")
	parseCmd.Flags().BoolVarP(&dumpAST, "ast", "a", false,
		"Dump the parsed AST to disk")
	parseCmd.Flags().BoolVarP(&genJSONForUI, "ui", "u", false,
		"Generate Win32 API JSON definitions for saferwall UI frontend.")
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

	logger := log.NewCustom("info").With(context.TODO())
	if _, err := os.Stat(includePath); os.IsNotExist(err) {
		logger.Errorf("The include directory does not exist ..")
		flag.Usage()
		os.Exit(0)
	}

	filePath := filepath.Join("assets", "header.h")
	code, err := utils.ReadAll(filePath)
	if err != nil {
		logger.Fatalf("reading header.h failed: %v", err)
	}

	config, err := cc.NewConfig(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		logger.Fatal(err)
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
	config.Predefined += "#define _MSC_FULL_VER 192930133\n"

	var sources []cc.Source
	sources = append(sources, cc.Source{Name: "<predefined>", Value: config.Predefined})
	sources = append(sources, cc.Source{Name: "<builtin>", Value: cc.Builtin})
	sources = append(sources, cc.Source{Name: "saferwall.c", Value: code})

	ast, err := cc.Translate(config, sources)
	if err != nil {
		logger.Fatalf("cc translate failed with:%v", err)
	}

	if dumpAST {
		r := strings.NewReader(ast.TranslationUnit.String())
		_, err = utils.WriteBytesFile("ast.txt", r)
		if err != nil {
			logger.Fatalf("failed to write ast: %v", err)
		}
	}

	// Use c-for-go to translate the AST to high level objects.
	myTranslator, err := translator.New(&translator.Config{})
	if err != nil {
		logger.Fatalf("failed to create new translator: %v", err)
	}
	myTranslator.Learn(ast)

	// Walk through all declarations and create list of APIs.
	var w32apis []entity.W32API
	for _, d := range myTranslator.Declares() {
		if strings.HasPrefix(d.Name, "__builtin_") {
			logger.Debugf("skipping builtin declaration: %s", d.Name)
			continue
		}

		var err error
		funcSpec, ok := d.Spec.(*translator.CFunctionSpec)
		if !ok {
			continue
		}

		var w32api entity.W32API

		retSpec, ok := funcSpec.Return.(*translator.CTypeSpec)
		if !ok {
			// We are dealing with functions that return type is void.
			w32api.RetType = "void"
		} else {
			// API return type.
			w32api.RetType = retSpec.Raw
			if retSpec.Raw == "" {
				w32api.RetType = retSpec.Base
			}
		}

		w32api.Name = d.Name
		w32api.DLL, err = utils.GetDLLName(d.Position.Filename, d.Name, sdkapiPath)
		if err != nil {
			logger.Debugf("failed to get the DLL name for: %s", d.Name)
			continue
		}

		funcDecl := ast.Scope.Nodes[d.Name][0].(*cc.Declarator)
		ft := funcDecl.Type().(*cc.FunctionType)

		w32api.Params = make([]entity.W32APIParam, len(funcSpec.Params))
		for idx, param := range funcSpec.Params {
			var w32apiParam entity.W32APIParam
			w32apiParam.Name = param.Name

			switch param.Spec.(type) {
			case *translator.CTypeSpec:
				paramSpec := param.Spec.(*translator.CTypeSpec)
				w32apiParam.Type = paramSpec.Raw
				if paramSpec.Raw == "" {
					w32apiParam.Type = paramSpec.Base
				}
			case *translator.CStructSpec:
				paramSpec := param.Spec.(*translator.CStructSpec)
				if paramSpec.Pointers > 1 {
					for i := uint8(1); i < paramSpec.Pointers; i++ {
						w32apiParam.Type = "*" + w32apiParam.Type
					}
				}

				w32apiParam.Type += paramSpec.Raw
			}

			paramDecl := ft.Parameters()[idx]
			if paramDecl.Declarator == nil {
				logger.Debugf("param declarator is nil for: %s", d.Name)
				w32api.Params[idx] = w32apiParam // even though incomplete
				continue
			}

			t := paramDecl.Declarator.Type()
			attr := t.Attributes()
			if attr != nil {
				attrVal := attr.AttrValue("anno")[0].(cc.StringValue)
				w32apiParam.Annotation = strings.Replace(string(attrVal), "\x00", "", -1)
			}
			w32api.Params[idx] = w32apiParam
		}

		w32apis = append(w32apis, w32api)
		logger.Info(w32api.String())
	}

	if genJSONForUI {

		// Read the list of APIs we are interested to hook.
		wantedAPIs, err := utils.ReadLines("./assets/hookapis.md")
		if err != nil {
			logger.Fatal(err)
		}

		uiMap := make(map[string][]map[string]string)
		for _, w32api := range w32apis {

			if !utils.StringInSlice(w32api.Name, wantedAPIs) {
				continue
			}

			params := make([]map[string]string, len(w32api.Params))
			for i, apiParam := range w32api.Params {
				params[i] = make(map[string]string)
				params[i]["typ"] = apiParam.Type
				params[i]["name"] = apiParam.Name
			}
			uiMap[w32api.Name] = params
		}

		for _, wantedAPI := range wantedAPIs {
			if _, ok := uiMap[wantedAPI]; !ok {
				logger.Infof("%s not found !", wantedAPI)
			}
		}

		data, _ := json.Marshal(uiMap)
		utils.WriteBytesFile("./assets/w32apis-ui.json", bytes.NewReader(data))
	}
}
