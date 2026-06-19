package admin

import (
	"github.com/ignisVeneficus/lumenta/tpl/data"
)

type MainPageContext struct {
	data.NavigationContext
	Cards []DashboardCard
}
type DashboardCard struct {
	ID           string // "images", "albums"
	TitleKey     string
	IconTitleKey string
	IconKey      string
	URL          string
	Stats        []DashboardStat
}

type DashboardStat struct {
	Label string
	Value string
}
