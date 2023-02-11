package main

import (
	"bytes"
	"flag"
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
	name    string
	methods []interfaceMethod
}

func (ir *interfaceResult) String() string {
	var buffer bytes.Buffer

	for i, method := range ir.methods {
		buffer.WriteString("  " + method.String())
		if i != len(ir.methods)-1 {
			buffer.WriteString("\n")
		}
	}

	return fmt.Sprintf(`%s {
%s
}`, ir.name, &buffer)
}

type interfaceMethod struct {
	name        string
	inputTypes  []string
	outputTypes []string
}

func (im *interfaceMethod) String() string {
	var buffer bytes.Buffer

	buffer.WriteString(im.name)

	buffer.WriteString("(")
	for i, input := range im.inputTypes {
		buffer.WriteString(input)
		if i != len(im.inputTypes)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString(")")

	buffer.WriteString("(")
	for i, output := range im.outputTypes {
		buffer.WriteString(output)
		if i != len(im.outputTypes)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString(")")
	return buffer.String()
}

type parseError struct {
	path string
	err  error
}

const Usage = `Usage: interface-inspector -code <path> -interface <interface name>
`

func main() {
	codeDirectory := flag.String("code", ".", "the directory that contains the code")
	interfaceName := flag.String("interface", "", "the name of the interface")

	flag.Parse()
	flag.Usage = func() {
		fmt.Printf("%s", Usage)
	}

	if *interfaceName == "" {
		flag.Usage()
		os.Exit(1)
	}

	var (
		interfaces = make([]interfaceResult, 0)

		structs    = make([]structResult, 0)
		structsMap = make(map[string]bool)
	)

	var (
		parseErrorsCh = make(chan parseError)
		ifacesCh      = make(chan interfaceResult)
		structCh      = make(chan structResult)
		readIfacesCh  = make(chan struct{})
	)

	// log parse errors
	go func() {
		for parseError := range parseErrorsCh {
			fmt.Printf("error parsing %s: %v\n", parseError.path, parseError.err)
		}
	}()

	// aggregate interfaces
	go func() {
		for iface := range ifacesCh {
			interfaces = append(interfaces, iface)
		}
		readIfacesCh <- struct{}{}
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
	var writeWg sync.WaitGroup
	err := filepath.Walk(*codeDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		writeWg.Add(1)
		go func() {
			defer writeWg.Done()
			err := parseFile(path, ifacesCh, structCh, *interfaceName)
			if err != nil {
				parseErrorsCh <- parseError{err: err, path: path}
			}
		}()
		return nil
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	writeWg.Wait()
	// close ifacesCh so the routine reading the interfaces can finish
	close(ifacesCh)
	close(structCh)
	// wait for the routine reading the interfaces to finish
	<-readIfacesCh

	if len(interfaces) == 0 {
		fmt.Printf("No interface found with name %s\n", *interfaceName)
		os.Exit(1)
	}

	fmt.Printf("Interfaces:\n")
	for _, iface := range interfaces {
		fmt.Println(iface.String())
	}

	fmt.Printf("Structs:\n")
	for _, strct := range structs {
		fmt.Println(strct.name)
	}
}

func parseFile(path string, interfaces chan<- interfaceResult, structs chan<- structResult, interfaceName string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		return err
	}

	ast.Inspect(f, func(node ast.Node) bool {
		if typeSpec, ok := node.(*ast.TypeSpec); ok {
			name := typeSpec.Name.Name
			// we have in interface
			theInterface, ok := typeSpec.Type.(*ast.InterfaceType)
			if ok && interfaceName == name {
				interfaces <- interfaceResult{name: name, methods: extractMethods(theInterface)}
				return true
			}

			// we have a struct
			theStruct, ok := typeSpec.Type.(*ast.StructType)
			if ok {
				structs <- structResult{name: name, strct: theStruct}
				return true
			}
		}
		return true
	})

	return nil
}

func extractMethods(iface *ast.InterfaceType) []interfaceMethod {
	methods := []interfaceMethod{}

	for _, field := range iface.Methods.List {
		method := interfaceMethod{}
		for _, name := range field.Names {
			method.name = name.Name
			typ := field.Type.(*ast.FuncType)
			method.inputTypes = getFieldTypes(typ.Params)
			method.outputTypes = getFieldTypes(typ.Results)
			methods = append(methods, method)
		}
	}
	return methods
}

func getFieldTypes(results *ast.FieldList) []string {
	outputTypes := []string{}
	if results != nil {
		for _, result := range results.List {
			outputTypes = append(outputTypes, result.Type.(*ast.Ident).Name)
		}
	}
	return outputTypes
}
