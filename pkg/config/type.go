package config

type Format string

const (
	FormatSQL Format = "sql"
	FormatGo  Format = "go"
)

func (f Format) String() string {
	return string(f)
}
