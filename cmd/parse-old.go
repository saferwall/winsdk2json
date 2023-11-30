// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/saferwall/winsdk2json/internal/parser"
	"github.com/saferwall/winsdk2json/internal/utils"
	"github.com/spf13/cobra"
)

// Used for flags.
var (
	sdkumPath      string
	hookapisPath   string
	customhookPath string
	printretval    bool
	printanno      bool
	minify         bool
)

func init() {

	parseCmdOld.Flags().StringVarP(&sdkumPath, "sdk", "", "C:\\Program Files (x86)\\Windows Kits\\10\\Include\\10.0.19041.0\\",
		"The path to the windows sdk directory")
	parseCmdOld.Flags().StringVarP(&sdkapiPath, "sdk-api", "", ".\\sdk-api",
		"The path to the sdk-api docs directory (https://github.com/MicrosoftDocs/sdk-api)")
	parseCmdOld.Flags().StringVarP(&hookapisPath, "hookapis", "", ".\\assets\\hookapis.md",
		"The path to a a text file thats defines which APIs to trace, new line separated.")
	parseCmdOld.Flags().StringVarP(&customhookPath, "customhookapis", "", ".\\assets\\custom_hook_apis.md",
		"The path to a a text file thats defines which APIs uses custom hook handlers")

	parseCmdOld.Flags().BoolVarP(&printretval, "printretval", "", false, "Print return value type for each API")
	parseCmdOld.Flags().BoolVarP(&printanno, "printanno", "", false, "Print list of annotation values")
	parseCmdOld.Flags().BoolVarP(&minify, "minify", "m", false, "Mininify json")
}

var parseCmdOld = &cobra.Command{
	Use:   "oldparse",
	Short: "Walk through the Windows SDK and parse the Win32 headers",
	Long:  `Walk through the Windows SDK and parse the Win32 headers to produce JSON files.`,
	Run: func(cmd *cobra.Command, args []string) {
		runOld()
	},
}

func runOld() {

	if sdkumPath == "" {
		flag.Usage()
		os.Exit(0)
	}

	if !utils.Exists(sdkumPath) {
		log.Fatal("Windows sdk directory does not exist")
	}
	if !utils.Exists(sdkapiPath) {
		log.Fatal("sdk-api directory does not exist")
	}
	if !utils.Exists(hookapisPath) {
		log.Fatal("hookapis.md does not exists")
	}
	if !utils.Exists(customhookPath) {
		log.Fatal("customhookapis.md does not exists")
	}

	// Read the list of APIs we are interested to hook.
	wantedAPIs, err := utils.ReadLines(hookapisPath)
	if err != nil {
		log.Fatalln(err)
	}
	if len(wantedAPIs) == 0 {
		log.Fatalln("hookapis.md is empty")
	}

	// Read the list of APIs we uses that uses a custom hook handler.
	customHookHHandlerAPIs, err := utils.ReadLines(customhookPath)
	if err != nil {
		log.Fatalln(err)
	}
	if len(customHookHHandlerAPIs) == 0 {
		log.Println("customhookapis.md is empty")
	}

	// Initialize built-in compiler data types.
	parser.InitBuiltInTypes()

	// Get the list of files in the Windows SDK.
	files, err := utils.WalkAllFilesInDir(sdkumPath)
	if err != nil {
		log.Fatalln(err)
	}
	files = append(files, ".\\assets\\custom-def.h")

	// Defines some vars.
	m := make(map[string]map[string]parser.API)
	var winStructsRaw []string
	var winStructs []parser.Struct
	funcPtrs := make([]string, 0)
	var interestingHeaders = []string{
		"\\fileapi.h", "\\processthreadsapi.h", "\\winreg.h", "\\bcrypt.h", "\\rpcdce.h",
		"\\winbase.h", "\\urlmon.h", "\\memoryapi.h", "\\tlhelp32.h", "\\debugapi.h",
		"\\handleapi.h", "\\heapapi.h", "\\winsvc.h", "\\wincrypt.h", "\\wow64apiset.h",
		"\\libloaderapi.h", "\\sysinfoapi.h", "\\synchapi.h", "\\winuser.h", "\\ioapiset.h",
		"\\winhttp.h", "\\minwinbase.h", "\\minwindef.h", "\\winnt.h", "\\shellapi.h", "\\shlwapi.h",
		"\\ntdef.h", "\\basetsd.h", "\\wininet.h", "winsock.h", "securitybaseapi.h", "winsock2.h",
		"\\ws2tcpip.h", "\\corecrt_wstring.h", "\\corecrt_malloc.h", "processenv.h",
		"stringapiset.h", "errhandlingapi.h", "custom-def.h",
	}

	parsedAPI := 0
	var foundAPIs []string
	for _, file := range files {
		foundHdr := false
		var prototypes []string
		file = strings.ToLower(file)
		for _, headerName := range interestingHeaders {
			if strings.HasSuffix(file, headerName) {
				foundHdr = true
				break
			}
		}

		if !foundHdr {
			continue
		}

		// Read Win32 include API headers.
		log.Printf("Processing %s ...\n", file)
		data, err := utils.ReadAll(file)
		if err != nil {
			log.Fatalln(err)
		}

		// Parse typedefs.
		parser.ParseTypedefs(data)

		// Start parsing all struct in header file.
		a, b := parser.GetAllStructs(data)
		winStructsRaw = append(winStructsRaw, a...)
		winStructs = append(winStructs, b...)

		// Extract all function pointers.
		functionPointers := parser.ParseFunctionPointers(string(data))
		funcPtrs = append(funcPtrs, functionPointers...)

		// Grab all API prototypes.
		// 1. Ignore: FORCEINLINE
		r := regexp.MustCompile(parser.RegAPIs)
		matches := r.FindAllString(string(data), -1)
		for _, v := range matches {
			prototype := utils.RemoveAnnotations(v)
			prototype = utils.StandardizeSpaces(prototype)
			prototype = utils.Standardize(prototype)
			prototypes = append(prototypes, prototype)

			if strings.Contains(v, "lstrcatA") {
				log.Print(v)
			}

			// Only parse APIs we want to hook.
			if strings.HasPrefix(prototype, "SHSTDAPI_(HINSTANCE)") {
				prototype = strings.ReplaceAll(prototype, "SHSTDAPI_(HINSTANCE)", "")
				prototype = "HINSTANCE SHSTDAPI" + prototype
			} else if strings.HasPrefix(prototype, "LWSTDAPI_(LPCSTR)") {
				prototype = strings.ReplaceAll(prototype, "LWSTDAPI_(LPCSTR)", "")
				prototype = "LPCSTR LWSTDAPI" + prototype
			} else if strings.HasPrefix(prototype, "LWSTDAPI_(BOOL)") {
				prototype = strings.ReplaceAll(prototype, "LWSTDAPI_(BOOL)", "")
				prototype = "BOOL LWSTDAPI" + prototype
			} else if strings.HasPrefix(prototype, "LWSTDAPI_(PCSTR)") {
				prototype = strings.ReplaceAll(prototype, "LWSTDAPI_(PCSTR)", "")
				prototype = "PCSTR LWSTDAPI" + prototype
			} else if strings.HasPrefix(prototype, "LWSTDAPI_(PCWSTR)") {
				prototype = strings.ReplaceAll(prototype, "LWSTDAPI_(PCWSTR)", "")
				prototype = "PCWSTR LWSTDAPI" + prototype
			} else if strings.Contains(prototype, "// deprecated: annotation is as good as it gets") {
				prototype = strings.ReplaceAll(prototype, "// deprecated: annotation is as good as it gets", "")

			}
			mProto := utils.RegSubMatchToMapString(parser.RegProto, prototype)
			if !utils.StringInSlice(mProto["ApiName"], wantedAPIs) && !utils.StringInSlice(mProto["ApiName"], customHookHHandlerAPIs) {
				continue
			}

			// Parse the API prototype.
			papi := parser.ParseAPI(prototype)

			// Find which DLL this API belongs to. Unfortunately, the sdk does
			// not give you this information, we look into the sdk-api markdown
			// docs instead. (Normally, we could have parsed everything from
			// the md files, but they are missing the parameters type!).
			dllname, err := utils.GetDLLName(file, papi.Name, sdkapiPath)
			if err != nil {
				log.Printf("Failed to get the DLL name for: %s", papi.Name)
				dllname = "ntdll.dll"
			}
			if _, ok := m[dllname]; !ok {
				m[dllname] = make(map[string]parser.API)
			}
			m[dllname][papi.Name] = papi
			parsedAPI++
			foundAPIs = append(foundAPIs, papi.Name)
		}

		// Write raw prototypes to a text file.
		if len(prototypes) > 0 {
			_, err = utils.WriteStrSliceToFile("./dump/prototypes-"+filepath.Base(file)+".inc", prototypes)
			if err != nil {
				log.Fatalf("Failed to dump prototype %s", filepath.Base(file)+".inc")
			}
		}
	}

	// Append the NTDLL definitions.
	ntdll_json, err := utils.ReadAll("./assets/ntdll.json")
	if err != nil {
		log.Fatalf("Failed to read NTDLL definitions,, err: %v", err)
	}
	ntdll_defs := make(map[string]map[string]parser.API)
	err = json.Unmarshal([]byte(ntdll_json), &ntdll_defs)
	if err != nil {
		log.Fatalf("Failed to unmarshall NTDLL definitions,, err: %v", err)
	}
	m["ntdll.dll"] = ntdll_defs["ntdll.dll"]

	// Marshall and write to json file.
	if len(m) > 0 {
		data, _ := json.MarshalIndent(m, "", " ")
		_, err = utils.WriteBytesFile("./assets/apis.json", bytes.NewReader(data))
		if err != nil {
			log.Fatalf("Failed to dump apis.json")
		}
	}

	// Write struct results.
	utils.WriteStrSliceToFile("./dump/winstructs.h", winStructsRaw)
	d, _ := json.MarshalIndent(winStructs, "", " ")
	utils.WriteBytesFile("./assets/structs.json", bytes.NewReader(d))

	if printretval {
		for dll, v := range m {
			log.Printf("DLL: %s\n", dll)
			log.Println("====================")
			for api, vv := range v {
				log.Printf("API: %s:%s() => %s\n", vv.CallingConvention, api, vv.ReturnValueType)
				if !utils.StringInSlice(api, wantedAPIs) {
					log.Printf("Not found")
				}
			}
		}
	}

	log.Printf("Parsed API count: %d, Wanted API Count: %d", parsedAPI, len(wantedAPIs))

	// Init custom types.
	parser.InitCustomTypes(winStructs)

	if printanno || minify {
		data, err := utils.ReadAll("./assets/apis.json")
		if err != nil {
			log.Fatalln(err)
		}
		apis := make(map[string]map[string]parser.API)
		err = json.Unmarshal(data, &apis)
		if err != nil {
			log.Fatalln(err)
		}

		if printanno {
			var annotations []string
			var types []string
			for _, v := range apis {
				for _, vv := range v {
					for _, param := range vv.Params {
						if !utils.StringInSlice(param.Annotation, annotations) {
							annotations = append(annotations, param.Annotation)
							// log.Println(param.Annotation)
						}

						if !utils.StringInSlice(param.Type, types) {
							types = append(types, param.Type)
							log.Println(param.Type)
						}
					}
				}
			}
		}

		if minify {
			// Minifi APIs.
			data, _ := json.Marshal(parser.MinifyAPIs(apis, customHookHHandlerAPIs))
			utils.WriteBytesFile("./assets/mini-apis.json", bytes.NewReader(data))

			// Minify Structs/Unions.
			data, _ = json.Marshal(parser.MinifyStructAndUnions(winStructs))
			utils.WriteBytesFile("./assets/mini-structs.json", bytes.NewReader(data))
		}
		os.Exit(0)
	}
}
