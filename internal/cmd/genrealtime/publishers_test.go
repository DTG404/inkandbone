package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func generatedPayloadLiteral(expression ast.Expr) bool {
	pointer, ok := expression.(*ast.UnaryExpr)
	if !ok || pointer.Op != token.AND {
		return false
	}
	literal, ok := pointer.X.(*ast.CompositeLit)
	if !ok {
		return false
	}
	name := ""
	switch value := literal.Type.(type) {
	case *ast.Ident:
		name = value.Name
	case *ast.SelectorExpr:
		name = value.Sel.Name
	}
	return strings.HasSuffix(name, "Payload")
}

func compositeName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return value.Sel.Name
	}
	return ""
}

func TestProductionPublishersUseGeneratedPayloadStructs(t *testing.T) {
	root, err := filepath.Abs("../../..")
	require.NoError(t, err)
	fset := token.NewFileSet()

	for _, directory := range []string{"internal/api", "internal/mcp"} {
		require.NoError(t, filepath.WalkDir(filepath.Join(root, directory), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "realtime_gen.go") {
				return walkErr
			}
			file, parseErr := parser.ParseFile(fset, path, nil, 0)
			require.NoError(t, parseErr)
			initializers := map[string]ast.Expr{}
			ast.Inspect(file, func(node ast.Node) bool {
				switch value := node.(type) {
				case *ast.AssignStmt:
					for index, left := range value.Lhs {
						if identifier, ok := left.(*ast.Ident); ok && index < len(value.Rhs) {
							initializers[identifier.Name] = value.Rhs[index]
						}
					}
				case *ast.ValueSpec:
					for index, name := range value.Names {
						if index < len(value.Values) {
							initializers[name.Name] = value.Values[index]
						}
					}
				}
				return true
			})

			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || compositeName(call.Fun) != "Publish" || len(call.Args) != 1 {
					return true
				}
				event, ok := call.Args[0].(*ast.CompositeLit)
				if !ok || compositeName(event.Type) != "Event" {
					return true
				}
				var payload ast.Expr
				for _, element := range event.Elts {
					field, ok := element.(*ast.KeyValueExpr)
					if ok && compositeName(field.Key) == "Payload" {
						payload = field.Value
					}
				}
				if identifier, ok := payload.(*ast.Ident); ok {
					payload = initializers[identifier.Name]
				}
				require.Truef(t, generatedPayloadLiteral(payload), "%s:%d Publish payload must use a generated payload struct", path, fset.Position(event.Pos()).Line)
				return true
			})
			return nil
		}))
	}
}

func TestBackendSSELiteralNamesBelongToContract(t *testing.T) {
	root, err := filepath.Abs("../../..")
	require.NoError(t, err)
	contractData, err := os.ReadFile(filepath.Join(root, "contracts/realtime.json"))
	require.NoError(t, err)
	var realtime contract
	require.NoError(t, json.Unmarshal(contractData, &realtime))
	allowed := map[string]bool{}
	for _, event := range realtime.SSE {
		allowed[event.Name] = true
	}

	fset := token.NewFileSet()
	require.NoError(t, filepath.WalkDir(filepath.Join(root, "internal"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "realtime_gen.go") {
			return walkErr
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, parseErr)
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok || compositeName(literal.Type) != "SSEEvent" {
				return true
			}
			for _, element := range literal.Elts {
				field, ok := element.(*ast.KeyValueExpr)
				value, literalValue := field.Value.(*ast.BasicLit)
				if !ok || compositeName(field.Key) != "Type" || !literalValue || value.Kind != token.STRING {
					continue
				}
				name, unquoteErr := strconv.Unquote(value.Value)
				require.NoError(t, unquoteErr)
				require.Truef(t, allowed[name], "%s:%d unknown SSE event %q", path, fset.Position(value.Pos()).Line, name)
			}
			return true
		})
		return nil
	}))
}

func TestFrontendHasNoHandwrittenGeneratedEventInterfaces(t *testing.T) {
	root, err := filepath.Abs("../../..")
	require.NoError(t, err)
	for _, relative := range []string{"web/src/JournalPanel.tsx", "web/src/CharacterSheetPanel.tsx"} {
		content, readErr := os.ReadFile(filepath.Join(root, relative))
		require.NoError(t, readErr)
		text := string(content)
		for _, duplicate := range []string{"SessionUpdatedEvent", "XPAddedEvent", "CharacterUpdatedEvent"} {
			require.NotContains(t, text, "interface "+duplicate)
		}
	}
}
