package controllers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// outboundRequestTypes are *Request structs that are sent to another service, never bound
// from a client, so they carry no sanitize tags.
var outboundRequestTypes = map[string]bool{
	"controllers.ollamaChatRequest": true, // body sent to Ollama
	"models.OllamaChatRequest":      true, // body sent to Ollama
}

// TestRequestStructsDeclareSanitization fails when a string field of a request struct has
// no `sanitize` tag. Every struct whose name ends in Request (in models/ and controllers/,
// function-local ones included) is checked, plus the struct types its fields nest. A field
// that must stay exactly as sent — a password, an image — says so with sanitize:"-".
// See docs/conventions.md → Input sanitisation.
func TestRequestStructsDeclareSanitization(t *testing.T) {
	structs := map[string]*ast.StructType{}
	collectStructs(t, "../models", "models", structs)
	collectStructs(t, ".", "controllers", structs)

	var queue []string
	for name := range structs {
		if strings.HasSuffix(name, "Request") && !outboundRequestTypes[name] {
			queue = append(queue, name)
		}
	}
	if len(queue) < 30 {
		t.Fatalf("found only %d request structs; is the parser looking in the right place?", len(queue))
	}

	var missing []string
	seen := map[string]bool{}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if seen[name] {
			continue
		}
		seen[name] = true
		pkg := strings.SplitN(name, ".", 2)[0]
		checkStructFields(name, pkg, structs[name], structs, &queue, &missing)
	}

	sort.Strings(missing)
	for _, field := range missing {
		t.Errorf("%s holds a string but has no sanitize tag (use sanitize:\"-\" to keep it as sent)", field)
	}
}

func collectStructs(t *testing.T, dir, pkg string, structs map[string]*ast.StructType) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fileSet, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			if spec, ok := node.(*ast.TypeSpec); ok {
				if structType, ok := spec.Type.(*ast.StructType); ok {
					structs[pkg+"."+spec.Name.Name] = structType
				}
			}
			return true
		})
	}
}

func checkStructFields(owner, pkg string, structType *ast.StructType, structs map[string]*ast.StructType, queue *[]string, missing *[]string) {
	for _, field := range structType.Fields.List {
		fieldName := "(embedded)"
		if len(field.Names) > 0 {
			if !field.Names[0].IsExported() {
				continue
			}
			fieldName = field.Names[0].Name
		}

		if holdsString(field.Type) {
			if !hasSanitizeTag(field) {
				*missing = append(*missing, owner+"."+fieldName)
			}
			continue
		}

		switch nested := baseType(field.Type).(type) {
		case *ast.StructType:
			checkStructFields(owner+"."+fieldName, pkg, nested, structs, queue, missing)
		case *ast.Ident:
			if _, ok := structs[pkg+"."+nested.Name]; ok {
				*queue = append(*queue, pkg+"."+nested.Name)
			}
		case *ast.SelectorExpr:
			if ident, ok := nested.X.(*ast.Ident); ok {
				if _, ok := structs[ident.Name+"."+nested.Sel.Name]; ok {
					*queue = append(*queue, ident.Name+"."+nested.Sel.Name)
				}
			}
		}
	}
}

// baseType strips pointers, slices and arrays.
func baseType(expr ast.Expr) ast.Expr {
	for {
		switch typed := expr.(type) {
		case *ast.StarExpr:
			expr = typed.X
		case *ast.ArrayType:
			expr = typed.Elt
		default:
			return expr
		}
	}
}

// holdsString reports string, *string, []string, *[]string and the like.
func holdsString(expr ast.Expr) bool {
	ident, ok := baseType(expr).(*ast.Ident)
	return ok && ident.Name == "string"
}

func hasSanitizeTag(field *ast.Field) bool {
	if field.Tag == nil {
		return false
	}
	raw, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return false
	}
	_, ok := reflect.StructTag(raw).Lookup("sanitize")
	return ok
}
