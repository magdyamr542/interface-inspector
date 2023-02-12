package main

import (
	"flag"
	"fmt"
	"go/ast"
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
	return fmt.Sprintf("%s %s:%d:%d", s.name, s.position.Filename, s.position.Line, s.position.Column)
}

const Usage = `Usage: interface-inspector [OPTIONS]

Options:
 package_dir	The directory that contains the package where the interface is defined
 package	The name of the package that the interface belongs to
 interface	The name of the interface

Example:
 interface-inspector -package_dir pkg/cmd -package cmd -interface Stringer	This will show all structs implementing the interface "Stringer".
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
		fmt.Printf("error: load packages: %v\n", err)
		os.Exit(1)
	}

	// search for the interface in the package
	iface, err := findInterface(pkgs, *packageName, *packageDirectory, *interfaceName)
	if err != nil {
		fmt.Printf("error: find interfaces: %v\n", err)
		os.Exit(1)
	}

	// find structs
	strcts := findStrcts(pkgs)
	strctsImplementingIface := getStrctsImplementingIface(*packageDirectory, strcts, iface)
	if len(strctsImplementingIface) == 0 {
		fmt.Printf("error: no structs implement the interface %q defined in package %q\n", *interfaceName, *packageName)
		os.Exit(1)
	}

	for _, strct := range strctsImplementingIface {
		fmt.Printf("%s\n", strct.String())
	}
}

// findInterface finds an interface with the name interfaceName in package packageName
func findInterface(pkgs []*packages.Package, packageName, packageDirectory, interfaceName string) (findInterfaceResult, error) {
	var astf []*ast.File
	pkgFound := false
	var thePackage *packages.Package
	var isRootDir = packageDirectory == "." || packageDirectory == "./"
	for _, pkg := range pkgs {
		if pkg.Name == packageName && (strings.Contains(pkg.PkgPath, packageDirectory) || isRootDir) {
			pkgFound = true
			thePackage = pkg
			for _, f := range pkg.Syntax {
				astf = append(astf, f)
			}
			break
		}
	}

	if !pkgFound {
		return findInterfaceResult{}, fmt.Errorf("couldn't find a package named %q in %q", packageName, packageDirectory)
	}

	scope := thePackage.Types.Scope()

	interfaceType := scope.Lookup(interfaceName)
	if interfaceType == nil {
		return findInterfaceResult{}, fmt.Errorf("no such interface %q in package %q", interfaceName, packageName)
	}

	theInterface, ok := interfaceType.Type().Underlying().(*types.Interface)
	if !ok {
		return findInterfaceResult{}, fmt.Errorf("no such interface %q in package %q", interfaceName, packageName)
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

// findStrcts finds all structs in the project.
func findStrcts(pkgs []*packages.Package) []strctFound {
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

	return strcts
}
