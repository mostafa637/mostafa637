package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
)

var limit = flag.Int("lines", 100, "maximum file lines")
var fnLimit = flag.Int("func-lines", 20, "maximum function lines")

func main() {
	flag.Parse()
	root := "internal/core/cpu/wasmjit"
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}
	fset := token.NewFileSet()
	bad := walk(root, fset)
	if bad > 0 {
		os.Exit(1)
	}
}

func walk(root string, fset *token.FileSet) int {
	bad := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".go" {
			return err
		}
		if checkFile(path, fset) {
			bad++
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return bad + 1
	}
	return bad
}

func checkFile(path string, fset *token.FileSet) bool {
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true
	}
	lines := fset.Position(file.End()).Line
	bad := lines > *limit || hasCGo(file)
	if lines > *limit {
		fmt.Printf("%s: %d lines (limit %d)\n", path, lines, *limit)
	}
	if hasCGo(file) {
		fmt.Printf("%s: CGo import\n", path)
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && checkFunc(path, fset, fn) {
			bad = true
		}
	}
	return bad
}

func checkFunc(path string, fset *token.FileSet, fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}
	lines := fset.Position(fn.End()).Line - fset.Position(fn.Pos()).Line + 1
	if lines <= *fnLimit {
		return false
	}
	fmt.Printf("%s: %s is %d lines (limit %d)\n", path, fn.Name.Name, lines, *fnLimit)
	return true
}

func hasCGo(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path.Value == `"C"` {
			return true
		}
	}
	return false
}
