package db

import "strings"

const tableTemplate = "{{table}}"

type (
	sqlTemplate string
	sqlTable    string
)

func (te sqlTemplate) sql(t sqlTable) string {
	return strings.ReplaceAll(string(te), tableTemplate, string(t))
}
