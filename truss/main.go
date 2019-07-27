package main

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/emicklei/proto"
	"github.com/iancoleman/strcase"
)

var (
	protoFolder    string
	distFolder     string
	templateFolder string
)

func main() {
	flag.StringVar(&protoFolder, "proto_folder", "proto", "")
	flag.StringVar(&distFolder, "dist_folder", "pkg/platform", "")
	flag.StringVar(&templateFolder, "template_folder", "templates/truss/pkg", "")
	flag.StringVar(&customProto.Project, "project", "appstore", "")
	flag.StringVar(&customProto.ServiceName, "service_name", "platform", "")
	flag.Parse()
	err := filepath.Walk(protoFolder, ScanProtoFolder)
	if err != nil {
		panic(err)
	}
	// ScanGoFolder(".")
}

var customProto = Proto{
	MessageMap: map[string]*Message{},
}

type Proto struct {
	Services    []Service
	Imports     []string
	Messages    []Message
	MessageMap  map[string]*Message
	Package     string // Package 目錄名稱, 自動產生
	Project     string // Project 指定 e.g. appstore, atw
	ServiceName string // ServiceName 預設 platform
}

type Field struct {
	Name     string
	Type     string
	Repeated bool
}

type Message struct {
	Name   string
	Fields []Field
}
type Service struct {
	Name string
	APIs []API
}

type API struct {
	Name        string
	RequestType string
	ReturnsType string
	Method      string
	Path        string
	Body        string
	PathParams  []string
}

var GetStructFromName = regexp.MustCompile(`(Create|Read|List|Update|Delete)(.*)`)

var funcMap = template.FuncMap{
	"HasPrefix":        strings.HasPrefix,
	"HasSuffix":        strings.HasSuffix,
	"ToUpper":          strings.ToUpper,
	"ToLower":          strings.ToLower,
	"ToCamel":          strcase.ToCamel,
	"ToLowerCamel":     strcase.ToLowerCamel,
	"ToSnake":          strcase.ToSnake,
	"ToScreamingSnake": strcase.ToScreamingSnake,
	"ToKebab":          strcase.ToKebab,
	"ToGoCamel": func(name string) string {
		ret := strcase.ToCamel(name)
		if strings.HasSuffix(ret, "Id") {
			ret = ret[:len(ret)-2] + "ID"
		} else if strings.HasSuffix(ret, "Url") {
			ret = ret[:len(ret)-3] + "URL"
		} else if strings.HasSuffix(ret, "Ip") {
			ret = ret[:len(ret)-2] + "IP"
		} else if strings.HasSuffix(ret, "Uuid") {
			ret = ret[:len(ret)-4] + "UUID"
		} else if strings.HasSuffix(ret, "Json") {
			ret = ret[:len(ret)-4] + "JSON"
		}
		return ret
	},
	"ToScreamingKebab": strcase.ToScreamingKebab,
	"GetMessage": func(key string) *Message {
		return customProto.MessageMap[key]
	},
	"GetStructFromName": func(name string) string {
		return GetStructFromName.ReplaceAllString(name, "$2")
	},
	"GoType": GoType,
}

func GoType(name string) string {
	switch name {
	case "google.protobuf.Timestamp":
		return "db.Timestamp"
	default:
		return name
	}
}

func ScanProtoFolder(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() || !strings.HasSuffix(path, ".proto") {
		return nil
	}
	reader, _ := os.Open(path)
	defer reader.Close()
	parser := proto.NewParser(reader)
	definition, err := parser.Parse()
	if err != nil {
		panic(err)
	}
	proto.Walk(definition,
		proto.WithService(handleService(&customProto)),
		proto.WithMessage(handleMessage(&customProto)),
	)
	return filepath.Walk(templateFolder, ScanTemplateFolder(customProto))
}

func ScanTemplateFolder(p Proto) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".tpl") {
			return nil
		}
		// fmt.Println(path, info.Size())
		tpl, err := template.New(info.Name()).Funcs(funcMap).ParseFiles(path)
		if err != nil {
			panic(err)
		}

		distPath := filepath.Join(distFolder, strings.TrimSuffix(strings.TrimPrefix(path, templateFolder+"/"), ".tpl"))
		if err := os.MkdirAll(filepath.Dir(distPath), os.ModePerm); err != nil {
			panic(err)
		}
		f, err := os.Create(distPath)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		p.Package = filepath.Base(filepath.Dir(distPath))
		if err := tpl.Execute(f, p); err != nil {
			panic(err)
		}
		return nil
	}
}

func handleMessage(p *Proto) func(*proto.Message) {
	return func(m *proto.Message) {
		customMessage := &Message{
			Name: m.Name,
		}
		for i := range m.Elements {
			el := m.Elements[i]
			f, ok := el.(*proto.NormalField)
			if ok {
				customMessage.Fields = append(customMessage.Fields, Field{
					Name:     f.Name,
					Type:     f.Type,
					Repeated: f.Repeated,
				})
			}
		}
		if p.MessageMap[m.Name] == nil {
			p.MessageMap[m.Name] = customMessage
			p.Messages = append(p.Messages, *customMessage)
		}
	}
}

func handleService(p *Proto) func(*proto.Service) {
	return func(s *proto.Service) {
		service := Service{
			Name: s.Name,
		}
		for i := range s.Elements {
			el, ok := s.Elements[i].(*proto.RPC)
			if ok {
				for i2 := range el.Elements {
					el2, ok := el.Elements[i2].(*proto.Option)
					if ok {
						method, path, pathParams := GetHttpMethodAndPath(el2.Constant.OrderedMap)
						api := API{
							Name:        el.Name,
							RequestType: el.RequestType,
							ReturnsType: el.ReturnsType,
							Method:      method,
							Path:        path,
							PathParams:  pathParams,
							Body:        GetHttpBody(el2.Constant.OrderedMap),
						}
						service.APIs = append(service.APIs, api)
					}
				}
			}
		}
		p.Services = append(p.Services, service)
	}
}

func GetHttpBody(lm proto.LiteralMap) (body string) {
	if el, ok := lm.Get("body"); ok {
		return el.Source
	}
	return ""
}

var pathParamsRegex = regexp.MustCompile(`{([a-z_A-Z]*)}`)

func GetHttpMethodAndPath(lm proto.LiteralMap) (method, path string, pathParams []string) {
	methods := []string{"post", "get", "put", "delete", "patch"}
	for i := range methods {
		m := methods[i]
		if el, ok := lm.Get(m); ok {
			tmpParams := pathParamsRegex.FindAllString(el.Source, -1)
			for i := range tmpParams {
				tmpParam := tmpParams[i]
				pathParams = append(pathParams, tmpParam[1:len(tmpParam)-1])
			}
			method = m
			path = pathParamsRegex.ReplaceAllString(el.Source, ":${1}")
			return
		}
	}
	return
}
