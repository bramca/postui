package postui

import "fmt"

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func getListItem(key string, value any) item {
	listItem := item{title: key}
	switch value := value.(type) {
	case string:
		listItem = item{title: key, desc: " " + value}
	case float64:
		listItem = item{title: key, desc: fmt.Sprintf(" %.1f", value)}
	case int:
		listItem = item{title: key, desc: fmt.Sprintf(" %d", value)}
	case bool:
		listItem = item{title: key, desc: fmt.Sprintf(" %t", value)}
	case map[string]string:
		description := ""
		for subKey := range value {
			description = fmt.Sprintf("%s %s", description, subKey)
		}
		listItem = item{title: key, desc: description}
	case map[string]any:
		description := ""
		for subKey := range value {
			description = fmt.Sprintf("%s %s", description, subKey)
		}
		listItem = item{title: key, desc: description}
	case []string:
		description := ""
		for _, strValue := range value {
			description = fmt.Sprintf("%s %s", description, strValue)
		}
		listItem = item{title: key, desc: description}
	case []any:
		description := ""
		for _, strValue := range value {
			description = fmt.Sprintf("%s %s", description, strValue)
		}
		listItem = item{title: key, desc: description}
	}

	return listItem
}
