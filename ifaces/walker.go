package ifaces

import "fmt"

// Walker walks something
type Walker interface {
	Walk() error
}

type walker struct {
	url string
}

func (w *walker) Walk() error {
	fmt.Printf("waling %s\n", w.url)
	return nil
}

func NewWalker(url string) walker {
	return walker{url: url}
}
