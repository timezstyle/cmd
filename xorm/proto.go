// Copyright 2017 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"
	"text/template"

	"github.com/go-xorm/core"
	"github.com/iancoleman/strcase"
)

var ProtoCounter = map[string]int{}

var (
	ProtoTmpl LangTmpl = LangTmpl{
		template.FuncMap{"Mapper": mapper.Table2Obj,
			"Type": protoTypeStr,
			"Add": func(i, j int) int {
				return i + j
			},
			"IncCounter": func(key string) int {
				ProtoCounter[key] = ProtoCounter[key] + 1
				return ProtoCounter[key]
			},
			"ClearCounter": func() string {
				ProtoCounter = map[string]int{}
				return ""
			},
			"HasSuffix":      strings.HasSuffix,
			"ToLowerCamel":   strcase.ToLowerCamel,
			"ToGoLowerCamel": ToGoLowerCamel,
		},
		nil,
		genProtoImports,
	}
)

func ToGoLowerCamel(s string) string {
	ret := strcase.ToLowerCamel(s)
	if strings.HasSuffix(ret, "Id") {
		ret = ret[:len(ret)-2] + "ID"
	} else if strings.HasSuffix(ret, "Url") {
		ret = ret[:len(ret)-3] + "URL"
	}
	return ret
}

func ToGoCamel(s string) string {
	ret := strcase.ToCamel(s)
	if strings.HasSuffix(ret, "Id") {
		ret = ret[:len(ret)-2] + "ID"
	} else if ret == "id" {
		ret = "ID"
	} else if strings.HasSuffix(ret, "Url") {
		ret = ret[:len(ret)-3] + "URL"
	}
	return ret
}

func protoTypeStr(col *core.Column) string {
	tp := col.SQLType
	name := strings.ToUpper(tp.Name)
	switch name {
	case core.Bit, core.TinyInt, core.SmallInt, core.MediumInt, core.Int, core.Integer, core.Serial:
		return "int32"
	case core.BigInt, core.BigSerial:
		return "int64"
	case core.Char, core.Varchar, core.TinyText, core.Text, core.MediumText, core.LongText:
		return "string"
	case core.Date, core.DateTime, core.Time, core.TimeStamp:
		return "google.protobuf.Timestamp"
	case core.Decimal, core.Numeric:
		return "double"
	case core.Real, core.Float:
		return "float"
	case core.Double:
		return "double"
	case core.TinyBlob, core.Blob, core.MediumBlob, core.LongBlob, core.Bytea, core.Json:
		return "bytes"
	case core.Bool:
		return "bool"
	default:
		return "string"
	}
}

func genProtoImports(tables []*core.Table) map[string]string {
	imports := make(map[string]string)

	for _, table := range tables {
		for _, col := range table.Columns() {
			switch protoTypeStr(col) {
			case "google.protobuf.Timestamp":
				imports["google/protobuf/timestamp.proto"] = "google/protobuf/timestamp.proto"
			}
		}
	}
	return imports
}
