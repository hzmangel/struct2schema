package main

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"
)

var (
	pattern = "@struct2schema"
	dbType  = flag.String("dbType", "sqlite3", "Database used for the generated SQL command, available choise: mysql/sqlite3")
)

// SchemaInfo - saves table infos
type SchemaInfo struct {
	TableName string
	Fields    []SchemaField
	LastIdx   int
}

// SchemaField - saves schema field
type SchemaField struct {
	Name      string
	ValueType string
}

func processFile(inputPath string, templateStr string) {
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

		// Generate SQL command via pre-defined template string
		t := template.Must(template.New("sqlCommand").Parse(templateStr))
		err := t.Execute(os.Stdout, schemaInfo)
		if err != nil {
			log.Println("executing template:", err)
		}
	}
}

func getTableInfo(decl ast.Decl, schemaInfo *SchemaInfo) (found bool) {
	genDecl, ok := decl.(*ast.GenDecl)

	// Skip nil or error nodes
	if !ok {
		return
	}
	if genDecl.Doc == nil {
		return
	}

	// table structure should have commant before its code block, or it will not be handled by this generate
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
				fieldLen := 0

				switch typeSpec.Type.(type) {
				case *ast.StructType:
					structSpec := typeSpec.Type.(*ast.StructType)
					for _, elem := range structSpec.Fields.List {
						newField := SchemaField{
							Name:      elem.Names[0].Name,
							ValueType: typeConvert(elem.Type.(*ast.Ident).Name),
						}

						schemaInfo.Fields = append(schemaInfo.Fields, newField)
						fieldLen++
					}
				}

				schemaInfo.LastIdx = fieldLen - 1
			}
		}
	}

	if schemaInfo.TableName == "" {
		return
	}

	found = true
	return
}

// Convert golang type to specified DB field type
// TODO: Convert to getting from file or generate tool
func typeConvert(golangFieldType string) (dbFieldType string) {
	switch golangFieldType {
	case "uint", "int":
		switch *dbType {
		case "sqlite3":
			dbFieldType = "INTEGER"
		case "mysql":
			dbFieldType = "INT"
		}
	case "uint8", "int8", "byte":
		switch *dbType {
		case "sqlite3":
			dbFieldType = "INTEGER"
		case "mysql":
			dbFieldType = "TINYINT"
		}
	case "uint16", "int16":
		switch *dbType {
		case "sqlite3":
			dbFieldType = "INTEGER"
		case "mysql":
			dbFieldType = "SMALLINT"
		}
	case "uint32", "int32", "rune":
		switch *dbType {
		case "sqlite3":
			dbFieldType = "INTEGER"
		case "mysql":
			dbFieldType = "INT"
		}
	case "uint64", "int64":
		switch *dbType {
		case "sqlite3":
			dbFieldType = "INTEGER"
		case "mysql":
			dbFieldType = "BIGINT"
		}

	case "float32", "float64":
		switch *dbType {
		case "sqlite3":
			dbFieldType = "REAL"
		case "mysql":
			dbFieldType = "FLOAT"
		}

	case "string":
		switch *dbType {
		case "sqlite3":
			dbFieldType = "TEXT"
		case "mysql":
			dbFieldType = "MEDIUMTEXT"
		}
	}
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

	const sqlTemplateStr = `
CREATE TABLE IF NOT EXISTS {{.TableName}} ( {{$lastIdx := .LastIdx}} {{ range $idx, $field := .Fields }}
  {{.Name}} {{.ValueType}}{{ if ne $lastIdx $idx }}, {{end}}
{{ end }} )
`

	for _, path := range os.Args[1:] {
		processFile(path, sqlTemplateStr)
	}
}
