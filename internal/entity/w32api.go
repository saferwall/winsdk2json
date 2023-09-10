// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package entity

import "fmt"

// W32APIParam represents a parameter of a Win32 API.
type W32APIParam struct {
	Annotation string `json:"anno,omitempty"`
	Type       string `json:"type"`
	Name       string `json:"name"`
}

// W32API represents information about a Win32 API.
type W32API struct {
	DLL               string        `json:"dll,omitempty"`  // DLL that exports the API.
	Attribute         string        `json:"attr,omitempty"` // Microsoft-specific attribute.
	CallingConvention string        `json:"cc,omitempty"`   // Calling Convention.
	Name              string        `json:"name"`           // Name of the API.
	RetType           string        `json:"ret_type"`       // Return value type.
	Params            []W32APIParam `json:"params"`         // API Arguments.
}

func (api *W32API) String() string {
	s := fmt.Sprintf("%s - %s %s (", api.DLL, api.RetType, api.Name)
	if len(api.Params) == 0 {
		s += "void )"
		return s
	}

	for _, p := range api.Params {
		s += fmt.Sprintf("%s %s %s, ", p.Annotation, p.Type, p.Name)
	}
	s = s[:len(s)-2] + ")"
	return s
}
