package db

import "strings"

type (
	sqlTemplate string
	sqlTable    string
)

func (te sqlTemplate) sql(t sqlTable) string {
	return strings.ReplaceAll(string(te), "{{table}}", string(t))
}
