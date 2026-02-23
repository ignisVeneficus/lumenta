package functions

import (
	"sort"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

func SortTagsByLocale(tags []*dbo.Tag, locale string) {

	tag, err := language.Parse(locale)
	if err != nil {
		tag = language.English // fallback
	}

	coll := collate.New(tag)

	sort.Slice(tags, func(i, j int) bool {
		return coll.CompareString(tags[i].Name, tags[j].Name) < 0
	})
}
