// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/saferwall/winsdk2json/internal/entity"
	log "github.com/saferwall/winsdk2json/internal/logger"
	"github.com/saferwall/winsdk2json/internal/utils"
	"github.com/xlab/c-for-go/translator"
	"modernc.org/cc/v4"
)

func translate(source []byte) []entity.W32API {

	logger := log.NewCustom("info").With(context.TODO())

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
	config.Predefined += "#define WIN32_LEAN_AND_MEAN\n"

	var sources []cc.Source
	sources = append(sources, cc.Source{Name: "<predefined>", Value: config.Predefined})
	sources = append(sources, cc.Source{Name: "<builtin>", Value: cc.Builtin})
	sources = append(sources, cc.Source{Name: "saferwall.c", Value: source})

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
				for i := uint8(1); i < paramSpec.Pointers; i++ {
					w32apiParam.Type = w32apiParam.Type + "*"
				}
			case *translator.CStructSpec:
				paramSpec := param.Spec.(*translator.CStructSpec)
				w32apiParam.Type = paramSpec.Raw
				for i := uint8(0); i < paramSpec.Pointers; i++ {
					w32apiParam.Type = w32apiParam.Type + "*"
				}
				if strings.HasPrefix(w32apiParam.Type, "LP") {
					w32apiParam.Type = w32apiParam.Type[:len(w32apiParam.Type)-1]
				}
				if len(paramSpec.Members) == 1 && paramSpec.Members[0].Name == "unused" {
					w32apiParam.Type = w32apiParam.Type[:len(w32apiParam.Type)-1]
				}

			case *translator.CEnumSpec:
				paramSpec := param.Spec.(*translator.CEnumSpec)
				w32apiParam.Type = paramSpec.Tag
			}

			paramDecl := ft.Parameters()[idx]
			if paramDecl.Declarator == nil {
				logger.Debugf("param declarator is nil for: %s", d.Name)
				w32api.Params[idx] = w32apiParam // even though incomplete
				continue
			}

			t := paramDecl.Declarator.Type()
			// if pointerType, ok := t.(*cc.PointerType); ok {
			// 	t = pointerType.Elem()
			// }

			attr := t.Attributes()
			if attr != nil {
				attrAnno := attr.AttrValue("anno")
				attrVal := attrAnno[0].(cc.StringValue)
				w32apiParam.Annotation = strings.Replace(string(attrVal), "\x00", "", -1)

				var annoSize string
				attrSize := attr.AttrValue("size")
				if len(attrSize) > 0 {
					attrValSize := attrSize[0].(cc.StringValue)
					annoSize = strings.Replace(string(attrValSize), "\x00", "", -1)

					attrCount := attr.AttrValue("count")
					if len(attrCount) > 0 {
						attrValSize := attrCount[0].(cc.StringValue)
						attrCount := strings.Replace(string(attrValSize), "\x00", "", -1)
						w32apiParam.Annotation = fmt.Sprintf("%s(%s,%s)", w32apiParam.Annotation, annoSize, attrCount)
					} else {
						w32apiParam.Annotation = fmt.Sprintf("%s(%s)", w32apiParam.Annotation, annoSize)
					}
				}

			}
			w32api.Params[idx] = w32apiParam
		}

		w32apis = append(w32apis, w32api)
		logger.Debug(w32api.String())
	}

	return w32apis
}
