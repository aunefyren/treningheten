package database

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// GORM leaves zero-valued fields out of a struct condition, so
// Where(&models.Gear{GormModel: models.GormModel{ID: id}}) with id == uuid.Nil — which
// uuid.Parse accepts from a request — has no id filter at all and matches the first or
// every row. Primary-key inline conditions (First(&gear, id)) and a primary key carried
// by the model value (Delete(&models.Gear{GormModel: …})) drop out the same way. Queries
// are written with explicit column conditions instead: Where("gear.id = ?", id). See
// docs/data-conventions.md.

// gormConditionMethods take conditions as arguments; a struct or map there is the trap.
var gormConditionMethods = map[string]bool{"Where": true, "Or": true, "Not": true}

// gormFinisherMethods take a destination first; anything after it is an inline condition.
var gormFinisherMethods = map[string]bool{
	"First": true, "Last": true, "Take": true, "Find": true, "Delete": true,
	"FirstOrInit": true, "FirstOrCreate": true,
}

// gormModelValueMethods read a primary key out of a model value that carries one.
var gormModelValueMethods = map[string]bool{"Model": true, "Delete": true}

// isStructOrMapCondition reports whether an argument is a struct or map literal (or a
// pointer to one), or a pointer to a variable — a slice literal for an IN list is fine.
func isStructOrMapCondition(argument ast.Expr) bool {
	if unary, isUnary := argument.(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
		if _, isIdent := unary.X.(*ast.Ident); isIdent {
			return true
		}
		argument = unary.X
	}
	literal, isLiteral := argument.(*ast.CompositeLit)
	if !isLiteral {
		return false
	}
	_, isSlice := literal.Type.(*ast.ArrayType)
	return !isSlice
}

// isFilledModelLiteral reports whether an argument is a struct literal with fields set,
// e.g. &models.Gear{GormModel: models.GormModel{ID: id}}; an empty &models.Gear{} only
// names the table.
func isFilledModelLiteral(argument ast.Expr) bool {
	if unary, isUnary := argument.(*ast.UnaryExpr); isUnary && unary.Op == token.AND {
		argument = unary.X
	}
	literal, isLiteral := argument.(*ast.CompositeLit)
	return isLiteral && len(literal.Elts) > 0
}

// Every non-test Go file in the module is scanned, since controllers/ also query
// database.Instance directly.
func TestNoStructConditions(t *testing.T) {
	root := ".."
	files := token.NewFileSet()
	scanned := 0

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(files, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		scanned++
		ast.Inspect(file, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall {
				return true
			}
			selector, isSelector := call.Fun.(*ast.SelectorExpr)
			if !isSelector {
				return true
			}
			method := selector.Sel.Name
			position := files.Position(call.Pos())

			if gormConditionMethods[method] {
				for _, argument := range call.Args {
					if isStructOrMapCondition(argument) {
						t.Errorf("%s: %s with a struct or map condition; zero-valued fields (a nil UUID) drop out of it — use an explicit column condition", position, method)
					}
				}
			}
			if gormFinisherMethods[method] && len(call.Args) > 1 {
				t.Errorf("%s: %s with an inline condition; a zero-valued id drops out of it — use Where with an explicit column condition", position, method)
			}
			if gormModelValueMethods[method] && len(call.Args) > 0 && isFilledModelLiteral(call.Args[0]) {
				t.Errorf("%s: %s on a model value with fields set; a zero primary key drops out of it — use Where with an explicit column condition", position, method)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("failed to scan the source: %v", err)
	}
	if scanned < 100 {
		t.Errorf("only %d files scanned; is the module root still %q?", scanned, root)
	}
}
