package locale

import (
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
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

func FormatNumber(n any, locale string) string {
	tag, err := language.Parse(locale)
	if err != nil {
		tag = language.English // fallback
	}

	p := message.NewPrinter(tag)
	switch n.(type) {
	case int, int8, int16, int32, int64:
		return p.Sprintf("%d", reflect.ValueOf(n).Int())

	case uint, uint8, uint16, uint32, uint64:
		return p.Sprintf("%d", reflect.ValueOf(n).Uint())
	}
	return "-"
}

func FormatTime(t time.Time, locale string) string {
	return t.Format("2006-01-02 15:04:05")
}

func FormatDuration(duration *time.Duration, locale string, i18n *i18n.Service) string {
	if duration == nil {
		return "-"
	}
	secStr := i18n.T(locale, "common.duration.second.short", nil)
	minStr := i18n.T(locale, "common.duration.minute.short", nil)
	hourStr := i18n.T(locale, "common.duration.hour.short", nil)
	dayStr := i18n.T(locale, "common.duration.day.short", nil)

	sec := int(duration.Seconds())

	if sec < 60 {
		return fmt.Sprintf("%d%s", sec, secStr)
	}

	min := sec / 60
	sec = sec % 60

	if min < 60 {
		return fmt.Sprintf("%d%s %d%s", min, minStr, sec, secStr)
	}

	h := min / 60
	min = min % 60

	if h < 24 {
		return fmt.Sprintf("%d%s %d%s", h, hourStr, min, minStr)
	}
	d := h / 24
	h = h % 24
	return fmt.Sprintf("%d%s %d%s", d, dayStr, h, hourStr)
}
