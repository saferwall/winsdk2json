// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package parser

import (
	"log"

	"regexp"
	"strings"

	"github.com/saferwall/winsdk2json/internal/utils"
)

const (
	// RegAPIs is a regex that extract API prototypes.
	RegAPIs = `(_Success_|HANDLE|RPCRTAPI|INTERNETAPI|WINHTTPAPI|BOOLAPI|BOOL|STDAPI|SHSTDAPI|LWSTDAPI|WINUSERAPI|WINBASEAPI|WINADVAPI|NTSTATUS|NTAPI|WINSOCK_API_LINKAGE|_Must_inspect_result_|BOOLEAN|int|errno_t|wchar_t\*)[\w\s\)\(,\[\]\!*+=&<>/|:]+;`

	// RegProto extracts API information.
	RegProto = `(?P<Attr>WINBASEAPI|WINADVAPI|WSAAPI|RPCRTAPI)?( )?(?P<RetValType>[A-Za-z_]+) (?P<CallConv>WINAPI|APIENTRY|WSAAPI|SHSTDAPI|LWSTDAPI|NTAPI|RPC_ENTRY) (?P<ApiName>[a-zA-Z0-9]+)( )?\((?P<Params>.*)\);`

	// RegAPIParams parses params.
	RegAPIParams = `(?P<Anno>_In_|IN|OUT|_In_opt_|_Inout_opt_|_Out_|_Inout_|_Out_opt_|_Outptr_opt_|_Reserved_|_Frees_ptr_opt_|_(O|o)ut[\w(),+ *]+|_In[\w()+]+|_When[\w() =,!*]+) (?P<Type>[\w *]+) (?P<Name>[*a-zA-Z0-9]+)`

	// RegParam extacts API parameters.
	RegParam = `, `
)

// APIParam represents a paramter of a Win32 API.
type APIParam struct {
	Annotation string `json:"anno"`
	Type       string `json:"type"`
	Name       string `json:"name"`
}

// API represents information about a Win32 API.
type API struct {
	Attribute         string     `json:"-"`        // Microsoft-specific attribute.
	CallingConvention string     `json:"callconv"` // Calling Convention.
	Name              string     `json:"name"`     // Name of the API.
	ReturnValueType   string     `json:"retVal"`   // Return value type.
	Params            []APIParam `json:"params"`   // API Arguments.
	CountParams       uint8      `json:"-"`        // Count of Params.
}

func parseAPIParameter(params string) APIParam {
	m := utils.RegSubMatchToMapString(RegAPIParams, params)
	apiParam := APIParam{
		Annotation: m["Anno"],
		Name:       m["Name"],
		Type:       m["Type"],
	}

	// move the `*` to the type.
	if strings.HasPrefix(apiParam.Name, "*") {
		apiParam.Name = apiParam.Name[1:]
		apiParam.Type += "*"
	}
	return apiParam
}

func ParseAPI(apiPrototype string) API {
	m := utils.RegSubMatchToMapString(RegProto, apiPrototype)
	api := API{
		Attribute:         m["Attr"],
		CallingConvention: m["CallConv"],
		Name:              m["ApiName"],
		ReturnValueType:   m["RetValType"],
	}

	// Treat the VOID case.
	if m["Params"] == " VOID " {
		api.CountParams = 0
		api.Params = []APIParam{}
		return api
	}

	if api.Name == "" || api.CallingConvention == "" {
		log.Printf("Failed to parse: %s", apiPrototype)
		return api
	}

	re := regexp.MustCompile(RegParam)
	split := re.Split(m["Params"], -1)
	for i := 0; i < len(split); i++ {
		v := split[i]

		// Quick hack: some API like CreateToolhelp32Snapshot don't have SAL
		// annotations, so assume its _In_,
		ss := strings.Split(utils.StandardizeSpaces(v), " ")
		if len(ss) == 2 {
			v = "_In_ " + v
		}

		if strings.Contains(v, "(") {
			if !utils.IsValid(v) {
				v = v + ", " + split[i+1]
				if !utils.IsValid(v) {
					utils.IsValid(v + split[i+1])
				}
				i++

			}
		}
		api.Params = append(api.Params, parseAPIParameter("_"+v))
		api.CountParams++
	}
	return api
}
