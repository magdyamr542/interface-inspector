## Interface inspector

See which `structs` implement a given `interface`.

#### Example:

- Given the following project structure, we have an interface `Fetcher` which is implemented by both the `awsFetcher struct` and `facebookFetcher` struct.
- Running `interface-inspector -package fetcher -interface Fetcher` would output all structs which implement the interface **Fetcher** defined in the package named **fetcher**.
- Output example:
  ```
  awsFetcher /home/tester/Documents/projects/interface-inspector/pkg/aws/aws.go:3:6
  facebookFetcher /home/tester/Documents/projects/interface-inspector/pkg/facebook/facebook.go:3:6
  ```
- Clicking with the mouse on the path in the output in an editor like `VSCode` would open the strcut directly which is handy in big projects where a lot of structs implement an interface and one wants to see all of them.

  ```
  ├── go.mod
  ├── go.sum
  ├── main.go
  ├── pkg
  │   ├── aws
  │   │   └── aws.go
  │   ├── facebook
  │   │   └── facebook.go
  │   └── fetcher
  │   └── fetcher.go
  └── README.md

  ```

  ```go
  // file: pkg/fetcher/fetcher.go
  package fetcher
  type Fetcher interface {
  	Fetch (url string) ([]byte , error)
  }
  ```

  ```go
  // file: pkg/aws/aws.go
  package aws
  type awsFetcher struct {
  }
  func (a *awsFetcher) Fetch (url string) ([]byte , error) {
  	return nil , nil
  }
  ```

  ```go
  // file: pkg/facebook/facebook.go
  package facebook
  type facebookFetcher struct {
  }
  func (a *facebookFetcher) Fetch (url string) ([]byte , error) {
  	return nil , nil
  }
  ```

#### Usage:

- Run `interface-inspector -h`

#### TODOS:

- Write a VSCode extension to interface with this. the extension should return the output in something like a quickpick list similar to what vscode does with the output of the language server.
