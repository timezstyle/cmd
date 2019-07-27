package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/davecgh/go-spew/spew"
)

func ScanGoFolder(path string) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, nil, 0)
	if err != nil {
		panic(err)
	}

	// var v visitor
	for i := range pkgs {
		pkg := pkgs[i]
		ast.Inspect(pkg, func(n ast.Node) bool {
			switch n.(type) {
			case *ast.InterfaceType:
				spew.Dump(n)
			}
			return true
		})
		// ast.Walk(v, pkg)
		for j := range pkg.Files {
			pkgFile := pkg.Files[j]
			for _, x := range pkgFile.Decls {
				if x, ok := x.(*ast.GenDecl); ok {
					if x.Tok != token.TYPE {
						continue
					}
					for _, x := range x.Specs {
						if x, ok := x.(*ast.TypeSpec); ok {
							iname := x.Name
							if x, ok := x.Type.(*ast.InterfaceType); ok {
								for _, x := range x.Methods.List {
									if len(x.Names) == 0 {
										continue
									}
									mname := x.Names[0].Name
									fmt.Println("interface:", iname, "method:", mname)

								}
							}
						}
					}
				}
			}
		}
	}

}

type visitor struct {
	locals map[string]int
}

func (v visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	switch t := n.(type) {
	case *ast.InterfaceType:
		spew.Dump(t)
		// // if t.Name.IsExported() {
		// switch t2 := t.Type.(type) {
		// // and are interfaces
		// case *ast.InterfaceType:
		// 	fmt.Println(t.Name.Name)
		// default:
		// 	fmt.Printf("%#v\n", t2)
		// }

		fmt.Printf("%#v\n", t)
		// }
	case *ast.Ident:
		// fmt.Println(t.Obj)
		if t.Obj != nil {
			// fmt.Printf("%#v\n", t.Obj)
			// fmt.Printf("%#v\n", t.Obj.Decl)
			// fmt.Printf("%#v\n\n", t.Name)
		}
	default:
		// fmt.Printf("%#v\n\n", t)
	}
	return v
}
