// Copyright 2021 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package parser

import (
	"log"
	"regexp"
)

var (
	regFunctionPtr = `typedef[\w\s]+\(WINAPI \*(?P<Name>\w+)\)\(`
)

func ParseFunctionPointers(data string) []string {

	var funcPtrs []string
	r := regexp.MustCompile(regFunctionPtr)
	matches := r.FindAllStringSubmatch(data, -1)
	for _, m := range matches {
		if len(m) > 0 {
			log.Println(m[1])
			funcPtrs = append(funcPtrs, m[1])
		}
	}

	return funcPtrs
}
