// implements the `returnFrom()` proposal using (very fragile) syntax rewriting
// and the existing implementations of panic() and recover().
//
// Known limitations:
// * you cannot returnFrom to name an enclosing generic function
// * you must specific the type parameter to returnFrom an enclosing generic call.
// * not very well tested yet, please report bugs.
//
// usage: go build rewrite.go && ./rewrite <example.go >transformed.go && go run transformed.go
package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/gcexportdata"
)

type nested struct {
	decl   *ast.FuncDecl
	call   *ast.CallExpr
	modify bool
}

func (n *nested) TypeString() string {
	if n.decl != nil {
		if n.decl.Recv != nil {
			return types.ExprString(n.decl.Recv.List[0].Names[0]) + "." + types.ExprString(n.decl.Name)

		}
		return types.ExprString(n.decl.Name)
	}
	noTypeParams, _, _ := strings.Cut(types.ExprString(n.call.Fun), "[")
	return noTypeParams
}

var builtins = `
type EarlyReturn struct {
	name string
	callId int
	inst any
}
func (er *EarlyReturn) Error() string {
	return "call to returnFrom(" + er.name+ ") outside of " + er.name
}
func returnFrom(...any)  {}
func goreturnfrom(name string, callId int, a any, panick any) {
	if panick != nil { panic(panick) }
	panic(&EarlyReturn{name, callId, a})
}

var callId int
`

func main() {

	if len(os.Args) <= 1 || os.Args[1][0] == '-' {
		fmt.Fprintln(os.Stderr, "Usage: go run github.com/ConradIrwin/return-from example.go > rewritten.go && go run rewritten.go")
		os.Exit(2)
	}
	input, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	sinput := strings.Replace(string(input), "runtime.EarlyReturn", "EarlyReturn", -1)

	fset := token.NewFileSet()
	imports := map[string]*types.Package{}
	config := types.Config{Importer: gcexportdata.NewImporter(fset, imports)}
	info := types.Info{Types: map[ast.Expr]types.TypeAndValue{}}

	file, err := parser.ParseFile(fset, os.Args[1], strings.NewReader(sinput+builtins), 0)
	if err != nil {
		log.Fatal(err)
	}
	_, err = config.Check("main", fset, []*ast.File{file}, &info)
	if err != nil {
		log.Fatal(err)
	}

	nesting := []*nested{}
	astutil.Apply(file, func(c *astutil.Cursor) bool {
		n := c.Node()

		if fd, ok := n.(*ast.FuncDecl); ok {
			nesting = append(nesting, &nested{decl: fd})
			return true
		}

		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		rf, ok := ce.Fun.(*ast.Ident)
		if !(ok && rf.Name == "returnFrom") {
			nesting = append(nesting, &nested{call: ce})
			return true
		}

		if len(ce.Args) == 0 {
			pos := fset.Position(rf.Pos())
			log.Fatal(pos.Filename + ":" + fmt.Sprint(pos.Line) + ":" + fmt.Sprint(pos.Column) + " returnFrom() with no arguments")
		}
		fn := ce.Args[0]
		found := false
		for i := len(nesting) - 1; i >= 0; i-- {
			search, _, _ := strings.Cut(types.ExprString(fn), "[")
			fmt.Fprintln(os.Stderr, "want", search)
			if nesting[i].TypeString() == search {
				nesting[i].modify = true
				found = true

				c.Replace(&ast.CallExpr{
					Fun: &ast.Ident{
						Name: "goreturnfrom",
					},
					Args: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.STRING,
							Value: `"` + types.ExprString(fn) + `"`,
						},
						&ast.Ident{Name: "localCallId"},
						&ast.CompositeLit{
							Type: &ast.Ident{Name: fmt.Sprintf("earlyreturn%v", i)},
							Elts: ce.Args[1:],
						},
						&ast.CallExpr{
							Fun: &ast.Ident{
								Name: "recover",
							},
						},
					},
				})
				break
			}
		}
		if !found {
			pos := fset.Position(rf.Pos())
			log.Fatal(pos.Filename + ":" + fmt.Sprint(pos.Line) + ":" + fmt.Sprint(pos.Column) + " returnFrom(" + types.ExprString(fn) + ") cannot happen outside of a matching declaration or call")
		}
		return true

	}, func(c *astutil.Cursor) bool {
		var body *ast.BlockStmt
		var typ *ast.FuncType
		var nc ast.Node

		n := c.Node()
		if ce, ok := n.(*ast.CallExpr); ok {
			ni := nesting[len(nesting)-1]
			if ni.call != ce {
				return true
			}
			nesting = nesting[:len(nesting)-1]
			if !ni.modify {
				return true
			}

			resultType := info.TypeOf(ce.Fun).(*types.Signature).Results().String()

			newCall, err := parser.ParseExpr("func () " + resultType + " { }()")
			if err != nil {
				log.Fatal(err)
			}

			typ = newCall.(*ast.CallExpr).Fun.(*ast.FuncLit).Type
			body = newCall.(*ast.CallExpr).Fun.(*ast.FuncLit).Body
			body.List = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ni.call}}}
			if resultType == "()" {
				body.List = []ast.Stmt{&ast.ExprStmt{ni.call}}
			}

			if err != nil {
				log.Fatal(err)
			}
			nc = newCall
		} else if fd, ok := n.(*ast.FuncDecl); ok {
			ni := nesting[len(nesting)-1]
			nesting = nesting[:len(nesting)-1]
			if !ni.modify {
				return true
			}
			body = fd.Body
			typ = fd.Type
		} else {
			return true
		}

		structFields := &ast.FieldList{List: []*ast.Field{}}
		declaration := &ast.TypeSpec{
			Name: &ast.Ident{Name: fmt.Sprintf("earlyreturn%v", len(nesting))},
			Type: &ast.StructType{Fields: structFields},
		}
		assignment := ""

		sl := "_, ok = "
		if typ.Results != nil && typ.Results.NumFields() > 0 {
			for i, item := range typ.Results.List {
				if item.Names == nil {
					item.Names = []*ast.Ident{{Name: fmt.Sprintf("ret%v", i)}}
				}

				structFields.List = append(structFields.List, item)
				assignment += item.Names[0].Name + " = _s." + item.Names[0].Name + "\n"
			}
			sl = "_s, ok := "
		}
		declStmt := &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{declaration}}}

		addDefer, err := parser.ParseExpr(`
		func () {
			localCallId := callId
			callId++
			defer func () {
				_r := recover()
				if _r == nil { return }
				_i, ok := _r.(*EarlyReturn)
				if !ok { panic(_r) }
				if _i.callId != localCallId { panic(_r) }
				` + sl + `_i.inst.(` + fmt.Sprintf("earlyreturn%v", len(nesting)) + `)
				if !ok { panic(_r) }
				` + assignment + `
			}()
		  }
		`)
		if err != nil {
			log.Fatal(err)
		}

		body.List = append(append([]ast.Stmt{declStmt}, addDefer.(*ast.FuncLit).Body.List[:]...), body.List...)

		if nc != nil {
			c.Replace(nc)
		}

		return true
	})

	format.Node(os.Stdout, fset, file)
}
