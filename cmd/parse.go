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

	log "github.com/saferwall/winsdk2json/internal/logger"
	"github.com/saferwall/winsdk2json/internal/utils"
	"github.com/spf13/cobra"
)

// Used for flags.
var (
	sdkapiPath   string
	includePath  string
	phntPath     string
	dumpAST      bool
	genJSONForUI bool
)

func init() {

	parseCmd.Flags().StringVarP(&includePath, "include", "i", "./winsdk/10.0.22000.0",
		"Path to the Windows Kits include directory")
	parseCmd.Flags().StringVarP(&sdkapiPath, "sdk-api", "", "./sdk-api",
		"The path to the sdk-api docs directory (https://github.com/MicrosoftDocs/sdk-api)")
	parseCmd.Flags().StringVarP(&phntPath, "phnt", "", "./phnt",
		"The path to the Native API header files for the System Informer project.")
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

	w32apis1 := translate(code)

	filePath = filepath.Join("assets", "header2.h")
	code, err = utils.ReadAll(filePath)
	if err != nil {
		logger.Fatalf("reading header2.h failed: %v", err)
	}

	w32apis2 := translate(code)

	var uniqueIDs []string
	for _, w32api := range w32apis1 {
		id := w32api.DLL + "-" + w32api.Name
		uniqueIDs = append(uniqueIDs, id)
	}

	// WinINET conflicts.
	for _, w32api := range w32apis2 {
		id := w32api.DLL + "-" + w32api.Name
		if !utils.StringInSlice(id, uniqueIDs) {
			w32apis1 = append(w32apis1, w32api)
		}
	}

	marshaled, err := json.MarshalIndent(w32apis1, "", "   ")
	if err != nil {
		logger.Fatal(err)
	}
	utils.WriteBytesFile("./assets/w32apis-full.json", bytes.NewReader(marshaled))

	if genJSONForUI {

		// Read the list of APIs we are interested to hook.
		wantedAPIs, err := utils.ReadLines("./assets/hookapis.md")
		if err != nil {
			logger.Fatal(err)
		}

		uiMap := make(map[string][][2]string)
		for _, w32api := range w32apis1 {

			if !utils.StringInSlice(w32api.Name, wantedAPIs) {
				continue
			}

			params := make([][2]string, len(w32api.Params))
			for i, apiParam := range w32api.Params {
				params[i][0] = apiParam.Type
				params[i][1] = apiParam.Name
			}
			uiMap[w32api.Name] = params
		}

		notFound := 0
		for _, wantedAPI := range wantedAPIs {
			if _, ok := uiMap[wantedAPI]; !ok {
				logger.Infof("%s not found !", wantedAPI)
				notFound++
			}
		}
		logger.Infof("APIs not found: %d", notFound)

		marshaled, err := json.MarshalIndent(uiMap, "", "   ")
		if err != nil {
			logger.Fatal(err)
		}
		utils.WriteBytesFile("./assets/w32apis-ui.json", bytes.NewReader(marshaled))
	}
}
