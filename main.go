package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sync"
)

type structResult struct {
	name  string
	strct *ast.StructType
}

type interfaceResult struct {
	name  string
	iface *ast.InterfaceType
}

type parseError struct {
	path string
	err  error
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <directory>")
		os.Exit(1)
	}
	dir := os.Args[1]

	var (
		interfaces    = make([]interfaceResult, 0)
		interfacesMap = make(map[string]bool)
		structs       = make([]structResult, 0)
		structsMap    = make(map[string]bool)
		parseErrors   = make(chan parseError)
		ifacesCh      = make(chan interfaceResult)
		structCh      = make(chan structResult)
	)

	// log errors
	go func() {
		for parseError := range parseErrors {
			fmt.Printf("error parsing %s: %v\n", parseError.path, parseError.err)
		}
	}()

	// aggregate interfaces
	go func() {
		for iface := range ifacesCh {
			if _, ok := interfacesMap[iface.name]; !ok {
				interfaces = append(interfaces, iface)
				interfacesMap[iface.name] = true
			}
		}
	}()

	// aggregate structs
	go func() {
		for strct := range structCh {
			if _, ok := structsMap[strct.name]; !ok {
				structs = append(structs, strct)
				structsMap[strct.name] = true
			}
		}
	}()

	// walk and emit paths
	var wg sync.WaitGroup
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			parseFile(path, parseErrors, ifacesCh, structCh)
		}()
		return nil
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	wg.Wait()

	fmt.Printf("Interfaces\n")
	for _, iface := range interfaces {
		fmt.Println(iface.name)
	}
	fmt.Printf("Structs\n")
	for _, strct := range structs {
		fmt.Println(strct.name)
	}
}

func parseFile(path string, parseErrors chan<- parseError, interfaces chan interfaceResult, structs chan structResult) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		parseErrors <- parseError{path: path, err: err}
	}

	ast.Inspect(f, func(node ast.Node) bool {
		if typeSpec, ok := node.(*ast.TypeSpec); ok {
			name := typeSpec.Name.Name
			theInterface, ok := typeSpec.Type.(*ast.InterfaceType)
			if ok {
				interfaces <- interfaceResult{name: name, iface: theInterface}
				return true
			}

			theStruct, ok := typeSpec.Type.(*ast.StructType)
			if ok {
				structs <- structResult{name: name, strct: theStruct}
				return true
			}
		}
		return true
	})

}

func implements(iface *ast.InterfaceType, strct *ast.StructType) bool {
	for _, field := range strct.Fields.List {
		if field.Names == nil {
			// The field is an anonymous field, so check if it implements the interface
			fieldType, ok := field.Type.(*ast.StructType)
			if !ok {
				// The field is not an *ast.Ident, so continue to the next field
				continue
			}

			if implements(iface, fieldType) {
				// The anonymous field implements the interface, so the struct implements the interface
				return true
			}
		}
	}

	for _, method := range iface.Methods.List {
		found := false
		for _, strctField := range strct.Fields.List {
			if strctField.Names[0].Name == method.Names[0].Name &&
				strctField.Type == method.Type {
				found = true
				break
			}
		}

		if !found {
			// The struct does not implement all the methods of the interface, so it does not implement the interface
			return false
		}
	}

	// The struct implements all the methods of the interface, so it implements the interface
	return true
}
