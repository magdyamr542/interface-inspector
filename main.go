package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

type findInterfaceResult struct {
	pkg       types.Package
	iface     *types.Interface
	ifaceName string
}

type strctFound struct {
	obj      types.Object
	strct    types.Struct
	name     string
	position token.Position
}

func (s *strctFound) String() string {
	return fmt.Sprintf("%s %s %s:%d:%d", s.name, s.strct.String(), s.position.Filename, s.position.Line, s.position.Column)
}

const Usage = `Usage: interface-inspector [OPTIONS]

Options:
 package_dir	The directory that contains the package where the interface is defined
 package	The name of the package that the interface belongs to
 interface	The name of the interface

Example:
 interface-inspector \
   -package_dir pkg/cmd \ 
   -package cmd \
   -interface Stringer		This will show all structs implementing the interface "Stringer".
				The interface "Stringer" belongs to package "cmd" whose files are in "pkg/cmd"
				The structs to be examined are all under path "pkg"`

func main() {
	packageDirectory := flag.String("package_dir", ".", "path of the package containing the interface")
	packageName := flag.String("package", "", "the package name")
	interfaceName := flag.String("interface", "", "the name of the interface")

	flag.Usage = func() {
		fmt.Println(Usage)
	}
	flag.Parse()

	if *interfaceName == "" || *packageName == "" {
		flag.Usage()
		os.Exit(1)
	}

	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, "./...")
	if err != nil {
		os.Exit(1)
	}

	// search for the interface in the package
	iface, err := findInterface(pkgs, *packageName, *packageDirectory, *interfaceName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// find structs
	strcts, err := findStrcts(pkgs)
	if err != nil {
		fmt.Printf("error while finding structs: %v\n", err)
		os.Exit(1)
	}

	theStrcts := getStrctsImplementingIface(*packageDirectory, strcts, iface)
	if len(theStrcts) == 0 {
		fmt.Printf("no structs implement the interface %q defined in package %q\n", *interfaceName, *packageName)
		os.Exit(1)
	}

	for _, strct := range theStrcts {
		fmt.Printf("%s\n", strct.String())
	}
}

// findInterface finds an interface with the name interfaceName in package packageName
func findInterface(pkgs []*packages.Package, packageName, packageDirectory, interfaceName string) (findInterfaceResult, error) {

	var astf []*ast.File
	pkgFound := false
	var thePackage *packages.Package
	for _, pkg := range pkgs {
		if pkg.Name == packageName && strings.Contains(pkg.PkgPath, packageDirectory) {
			pkgFound = true
			thePackage = pkg
			for _, f := range pkg.Syntax {
				astf = append(astf, f)
			}
			break
		}
	}

	if !pkgFound {
		return findInterfaceResult{}, fmt.Errorf("couldn't find a package named %s in %s", packageName, packageDirectory)
	}

	scope := thePackage.Types.Scope()

	interfaceType := scope.Lookup(interfaceName)
	if interfaceType == nil {
		return findInterfaceResult{}, fmt.Errorf("no such interface %s in package %s", interfaceName, packageName)
	}

	theInterface, ok := interfaceType.Type().Underlying().(*types.Interface)
	if !ok {
		return findInterfaceResult{}, fmt.Errorf("no such interface %s in package %s", interfaceName, packageName)
	}

	return findInterfaceResult{pkg: *thePackage.Types, iface: theInterface, ifaceName: interfaceName}, nil
}

// getStrctsImplementingIface returns all structs from strcts that implement the interface iface
func getStrctsImplementingIface(path string, strcts []strctFound, iface findInterfaceResult) []strctFound {
	strctResult := make([]strctFound, 0)
	for _, strct := range strcts {
		ptr := types.NewPointer(strct.obj.Type())
		if types.Implements(ptr, iface.iface) {
			strctResult = append(strctResult, strct)
		}
	}

	return strctResult
}

// findStructsInDir finds all structs in directory dir.
func findStructsInDir(dir string) ([]*strctFound, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.AllErrors)
	if err != nil {
		return []*strctFound{}, nil
	}

	var astf []*ast.File
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			astf = append(astf, f)
		}
	}

	config := &types.Config{
		Error: func(e error) {
			fmt.Println(e)
		},
		Importer: importer.Default(),
	}

	info := types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pkg, err := config.Check(dir, fset, astf, &info)
	if err != nil {
		return []*strctFound{}, fmt.Errorf("error config.Check: %v", err)
	}

	scope := pkg.Scope()
	strcts := make([]*strctFound, 0)
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		theStruct, ok := obj.Type().Underlying().(*types.Struct)

		if ok {
			strcts = append(strcts, &strctFound{
				obj:      obj,
				strct:    *theStruct,
				name:     obj.Name(),
				position: fset.Position(obj.Pos())})
		}
	}
	return strcts, nil
}

// findStrcts finds all structs in the project under the path.
// it emits the found structs to structsCh and any error to errorsCh.
func findStrcts(pkgs []*packages.Package) ([]strctFound, error) {
	strcts := make([]strctFound, 0)
	for _, pkg := range pkgs {
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			theStruct, ok := obj.Type().Underlying().(*types.Struct)

			if ok {
				strcts = append(strcts, strctFound{
					obj:      obj,
					strct:    *theStruct,
					name:     obj.Name(),
					position: pkg.Fset.Position(obj.Pos())})
			}
		}

	}

	return strcts, nil
}
