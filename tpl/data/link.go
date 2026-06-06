package data

import "html/template"

type Link struct {
	URL      string
	Label    string
	LabelKey string
	LabelMap map[string]interface{}
	Title    string
	TitleKey string
	TitleMap map[string]interface{}
}

func (l Link) TemplateURL() template.URL {
	return template.URL(l.URL)
}
