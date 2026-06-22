package merge

import (
	"bytes"
	gostd "go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
)

func appendGo(existing, generated []byte) ([]byte, bool) {
	fset := token.NewFileSet()
	oldF, err := parser.ParseFile(fset, "existing.go", existing, parser.ParseComments)
	if err != nil {
		return nil, false
	}
	newF, err := parser.ParseFile(fset, "generated.go", generated, parser.ParseComments)
	if err != nil {
		return nil, false
	}
	if oldF.Name == nil || newF.Name == nil || oldF.Name.Name != newF.Name.Name {
		return nil, false
	}

	haveImport := map[string]bool{}
	for _, imp := range oldF.Imports {
		haveImport[importKey(imp)] = true
	}
	var importsToAdd []*gostd.ImportSpec
	for _, imp := range newF.Imports {
		k := importKey(imp)
		if !haveImport[k] {
			haveImport[k] = true
			importsToAdd = append(importsToAdd, imp)
		}
	}

	haveFunc := map[string]bool{}
	for _, decl := range oldF.Decls {
		if fn, ok := decl.(*gostd.FuncDecl); ok && fn.Name != nil {
			haveFunc[fn.Name.Name] = true
		}
	}
	var funcsToAdd []gostd.Decl
	for _, decl := range newF.Decls {
		fn, ok := decl.(*gostd.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		if haveFunc[fn.Name.Name] {
			continue
		}
		haveFunc[fn.Name.Name] = true
		funcsToAdd = append(funcsToAdd, fn)
	}

	if len(importsToAdd) == 0 && len(funcsToAdd) == 0 {
		return existing, true
	}

	for _, imp := range importsToAdd {
		gostd.SortImports(fset, oldF)
		_ = imp
		addImport(oldF, imp)
	}
	oldF.Decls = append(oldF.Decls, funcsToAdd...)

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, oldF); err != nil {
		return nil, false
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		out = buf.Bytes()
	}
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	return out, true
}

func importKey(imp *gostd.ImportSpec) string {
	name := ""
	if imp.Name != nil {
		name = imp.Name.Name
	}
	path := ""
	if imp.Path != nil {
		path = imp.Path.Value
	}
	return name + " " + path
}

func addImport(f *gostd.File, imp *gostd.ImportSpec) {
	for i, decl := range f.Decls {
		gd, ok := decl.(*gostd.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		gd.Specs = append(gd.Specs, imp)
		gd.Lparen = token.Pos(1)
		gd.Rparen = token.Pos(1)
		f.Decls[i] = gd
		f.Imports = append(f.Imports, imp)
		return
	}
	gd := &gostd.GenDecl{
		Tok:    token.IMPORT,
		Lparen: token.Pos(1),
		Rparen: token.Pos(1),
		Specs:  []gostd.Spec{imp},
	}
	f.Decls = append([]gostd.Decl{gd}, f.Decls...)
	f.Imports = append(f.Imports, imp)
}
