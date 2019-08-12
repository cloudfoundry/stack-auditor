package resources

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
)

type App struct {
	Org   string `json:"org"`
	Space string `json:"space"`
	Name  string `json:"name"`
	Stack string `json:"stack"`
	State string `json:"state"`
}

type Apps []App

func (a Apps) String() string {
	var list []string

	for _, app := range a {
		list = append(list, fmt.Sprintf("%s", app))
	}

	return strings.Join(list, "\n") + "\n"

}

func (a Apps) Len() int {
	return len(a)
}

func (a Apps) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}

func (a Apps) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a Apps) CSV() (string, error) {
	records := a.records()

	var buff bytes.Buffer

	w := csv.NewWriter(&buff)
	if err := w.WriteAll(records); err != nil {
		return "", err
	}

	return buff.String(), nil
}

func (a App) String() string {
	return fmt.Sprintf("%s/%s/%s %s %s", a.Org, a.Space, a.Name, a.Stack, a.State)
}

func (a Apps) headers() []string {
	return []string{"org", "space", "name", "stack", "state"}
}

func (a Apps) values() [][]string {
	var result [][]string
	for _, app := range a {
		result = append(result, []string{app.Org, app.Space,
			app.Name, app.Stack, app.State})
	}

	return result
}

func (a Apps) records() [][]string {
	var result [][]string

	headers := a.headers()
	values := a.values()

	result = append(result, headers)
	result = append(result, values...)

	return result
}
