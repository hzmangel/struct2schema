package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

var pattern = "@struct2schema"

// SchemaInfo - saves table infos
type SchemaInfo struct {
	TableName string
	Fields    []SchemaField
}

// SchemaField - saves schema field
type SchemaField struct {
	Name      string
	ValueType string
}

func processFile(inputPath string) {
	log.Printf("Processing file %s", inputPath)
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	// ast.Print(fset, f)

	var schemaInfo SchemaInfo

	for _, decl := range f.Decls {
		schemaInfo.Fields = []SchemaField{}
		var ok bool

		ok = getTableInfo(decl, &schemaInfo)
		if !ok {
			continue
		}

		log.Println(schemaInfo)
	}
}

func getTableInfo(decl ast.Decl, schemaInfo *SchemaInfo) (found bool) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}
	if genDecl.Doc == nil {
		return
	}

	tableStructFound := false
	for _, comment := range genDecl.Doc.List {
		if strings.Contains(comment.Text, pattern) {
			tableStructFound = true
			break
		}
	}

	if !tableStructFound {
		return
	}

	for _, spec := range genDecl.Specs {
		switch spec.(type) {
		case *ast.TypeSpec:
			schemaInfo.TableName, found = getTableName(spec)

			if found == true {
				typeSpec := spec.(*ast.TypeSpec)

				switch typeSpec.Type.(type) {
				case *ast.StructType:
					structSpec := typeSpec.Type.(*ast.StructType)
					for _, elem := range structSpec.Fields.List {
						newField := SchemaField{
							Name:      elem.Names[0].Name,
							ValueType: elem.Type.(*ast.Ident).Name,
						}
						schemaInfo.Fields = append(schemaInfo.Fields, newField)
					}
				}
			}
		}
	}

	if schemaInfo.TableName == "" {
		return
	}

	found = true
	return
}

func getTableName(spec ast.Spec) (tableName string, ok bool) {
	typeSpec := spec.(*ast.TypeSpec)
	if typeSpec.Name != nil {
		ok = true
		tableName = typeSpec.Name.Name
	}
	return
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("struct2schema: ")

	for _, path := range os.Args[1:] {
		processFile(path)
	}
}
