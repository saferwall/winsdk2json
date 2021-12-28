// Copyright 2021 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/saferwall/winsdk2json/pkg/parser"
	"github.com/saferwall/winsdk2json/pkg/utils"
)

const (
	// RegDllName extracts DLL name from markdown spec.
	RegDllName = `req\.dll: (?P<DLL>[\w]+\.dll)`
)

func removeAnnotations(apiPrototype string) string {
	apiPrototype = strings.Replace(apiPrototype, "_Must_inspect_result_", "", -1)
	apiPrototype = strings.Replace(apiPrototype, "__drv_aliasesMem", "", -1)
	apiPrototype = strings.Replace(apiPrototype, "_Success_(return != 0 && return < nBufferLength)", "", -1)
	apiPrototype = strings.Replace(apiPrototype, "_Success_(return != 0 && return < cchBuffer)", "", -1)
	apiPrototype = strings.Replace(apiPrototype, "_Success_(return != FALSE)", "", -1)
	apiPrototype = strings.Replace(apiPrototype, "_Ret_maybenull_", "", -1)
	apiPrototype = strings.Replace(apiPrototype, "_Post_writable_byte_size_(dwSize)", "", -1)
	apiPrototype = strings.Replace(apiPrototype, "_Post_ptr_invalid_", "", -1)
	apiPrototype = strings.Replace(apiPrototype, "__out_data_source(FILE)", "", -1)
	apiPrototype = strings.Replace(apiPrototype, " OPTIONAL", "", -1)

	return apiPrototype
}

func standardize(s string) string {
	if strings.HasPrefix(s, "BOOLAPI") {
		s = strings.Replace(s, "BOOLAPI", "BOOL WINAPI", -1)
	} else if strings.HasPrefix(s, "INTERNETAPI_(HINTERNET)") {
		s = strings.Replace(s, "INTERNETAPI_(HINTERNET)", "HINTERNET WINAPI", -1)
	} else if strings.HasPrefix(s, "INTERNETAPI_(DWORD)") {
		s = strings.Replace(s, "INTERNETAPI_(DWORD)", "DWORD WINAPI", -1)
	} else if strings.HasPrefix(s, "STDAPI") {
		s = strings.Replace(s, "STDAPI", "HRESULT WINAPI", -1)
	}
	return s
}

func getDLLName(file, apiname, sdkpath string) (string, error) {
	cat := strings.TrimSuffix(filepath.Base(file), ".h")
	functionName := "nf-" + cat + "-" + strings.ToLower(apiname) + ".md"
	mdFile := path.Join(sdkpath, "sdk-api-src", "content", cat, functionName)
	mdFileContent, err := utils.ReadAll(mdFile)
	if err != nil {
		log.Printf("Failed to find file: %s", mdFile)
		return "", err
	}
	m := utils.RegSubMatchToMapString(RegDllName, string(mdFileContent))
	return strings.ToLower(m["DLL"]), nil
}

func main() {

	// Parse arguments.
	sdkumPath := flag.String("sdk", "", "The path to the windows sdk directory i.e C:\\Program Files (x86)\\Windows Kits\\10\\Include\\10.0.19041.0\\")
	sdkapiPath := flag.String("sdk-api", "C:\\Code\\sdk-api", "The path to the sdk-api docs directory (https://github.com/MicrosoftDocs/sdk-api)")
	hookapisPath := flag.String("hookapis", "..\\assets\\hookapis.md", "The path to a a text file thats defines which APIs to trace, new line separated.")
	printretval := flag.Bool("printretval", false, "Print return value type for each API")
	printanno := flag.Bool("printanno", false, "Print list of annotation values")
	minify := flag.Bool("minify", false, "Mininify json")

	flag.Parse()

	if *sdkumPath == "" {
		flag.Usage()
		os.Exit(0)
	}

	if !utils.Exists(*sdkumPath) {
		log.Fatal("Windows sdk directory does not exist")
	}

	if !utils.Exists(*sdkapiPath) {
		log.Fatal("sdk-api directory does not exist")
	}
	if !utils.Exists(*hookapisPath) {
		log.Fatal("hookapis.md does not exists")
	}

	// Read the list of APIs we are interested to hook.
	wantedAPIs, err := utils.ReadLines(*hookapisPath)
	if err != nil {
		log.Fatalln(err)
	}
	if len(wantedAPIs) == 0 {
		log.Fatalln("hookapis.md is empty")
	}

	// Initialize built-in compiler data types.
	parser.InitBuiltInTypes()

	// Get the list of files in the Windows SDK.
	files, err := utils.WalkAllFilesInDir(*sdkumPath)
	if err != nil {
		log.Fatalln(err)
	}

	// Defines some vars.
	m := make(map[string]map[string]parser.API)
	var winStructsRaw []string
	var winStructs []parser.Struct
	funcPtrs := make([]string, 0)
	var interestingHeaders = []string{
		"\\fileapi.h", "\\processthreadsapi.h", "\\winreg.h", "\\bcrypt.h",
		"\\winbase.h", "\\urlmon.h", "\\memoryapi.h", "\\tlhelp32.h", "\\debugapi.h", "\\handleapi.h", "\\heapapi.h", "\\winsvc.h",
		"\\libloaderapi.h", "\\sysinfoapi.h", "\\synchapi.h", "\\winuser.h",
		"\\winhttp.h", "\\minwinbase.h", "\\minwindef.h", "\\winnt.h",
		"\\ntdef.h", "\\basetsd.h", "\\wininet.h", "winsock.h", "securitybaseapi.h", "winsock2.h",
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
			prototype := removeAnnotations(v)
			prototype = utils.StandardizeSpaces(prototype)
			prototype = standardize(prototype)
			prototypes = append(prototypes, prototype)

			// Only parse APIs we want to hook.
			mProto := utils.RegSubMatchToMapString(parser.RegProto, prototype)
			if strings.Contains(v, "HeapFree") {
				log.Print()
			}
			if !utils.StringInSlice(mProto["ApiName"], wantedAPIs) {
				continue
			}

			// Parse the API prototype.
			papi := parser.ParseAPI(prototype)

			// Find which DLL this API belongs to. Unfortunately, the sdk does
			// not give you this information, we look into the sdk-api markdown
			// docs instead. (Normally, we could have parsed everything from
			// the md files, but they are missing the parameters type!).
			dllname, err := getDLLName(file, papi.Name, *sdkapiPath)
			if err != nil {
				log.Printf("Failed to get the DLL name for: %s", papi.Name)
				continue
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
			utils.WriteStrSliceToFile("dump/prototypes-"+filepath.Base(file)+".inc", prototypes)
		}
	}

	// Marshall and write to json file.
	if len(m) > 0 {
		data, _ := json.MarshalIndent(m, "", " ")
		utils.WriteBytesFile("json/apis.json", bytes.NewReader(data))
	}

	// Write struct results.
	utils.WriteStrSliceToFile("dump/winstructs.h", winStructsRaw)
	d, _ := json.MarshalIndent(winStructs, "", " ")
	utils.WriteBytesFile("json/structs.json", bytes.NewReader(d))

	if *printretval {
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

	if *printanno || *minify {
		data, err := utils.ReadAll("json/apis.json")
		if err != nil {
			log.Fatalln(err)
		}
		apis := make(map[string]map[string]parser.API)
		err = json.Unmarshal(data, &apis)
		if err != nil {
			log.Fatalln(err)
		}

		if *printanno {
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

		if *minify {
			// Minifi APIs.
			data, _ := json.Marshal(parser.MinifyAPIs(apis))
			utils.WriteBytesFile("json/mini-apis.json", bytes.NewReader(data))

			// Minify Structs/Unions.
			data, _ = json.Marshal(parser.MinifyStructAndUnions(winStructs))
			utils.WriteBytesFile("json/mini-structs.json", bytes.NewReader(data))
		}
		os.Exit(0)
	}
}
