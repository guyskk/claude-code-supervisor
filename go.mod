module github.com/guyskk/ccc

go 1.25

require (
	github.com/google/uuid v1.6.0
	github.com/kaptinlin/jsonrepair v0.2.6
	github.com/schlunsen/claude-agent-sdk-go v0.5.1
	github.com/stretchr/testify v1.11.1
	github.com/twpayne/go-expect v0.0.2-0.20241130000624-916db2914efd
	github.com/xeipuuv/gojsonschema v1.2.0
)

// Replace with guyskk's fork that has structured outputs support
replace github.com/schlunsen/claude-agent-sdk-go => github.com/guyskk/claude-agent-sdk-go v0.0.0-20260109144502-ba5ff3a9b5ff

require (
	github.com/creack/pty/v2 v2.0.0-20231209135443-03db72c7b76c // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	golang.org/x/sys v0.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
