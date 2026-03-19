package data

import (
	"html/template"
)

type Folder struct {
	URL         template.URL
	Image       *uint64
	Name        string
	Description string
	Info        string
}

type Folders struct {
	Cards  []Folder
	Paging Paging
}

type FolderPageContext struct {
	ImageGridPageContext
	PageCards Folders
	Map       MultiMap
}

func CreateFolders[T any](items []T, paging Paging, mapper func(T) Folder) Folders {
	cards := make([]Folder, 0, len(items))
	for _, item := range items {
		cards = append(cards, mapper(item))
	}

	return Folders{
		Cards:  cards,
		Paging: paging,
	}
}

type MultiMap struct {
	APIURL      string
	Cluster     bool
	Popup       bool
	Hover       bool
	NrMaxPoints uint64
}
