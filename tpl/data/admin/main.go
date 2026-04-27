package admin

import (
	"github.com/ignisVeneficus/lumenta/tpl/data"
)

type MainPageContext struct {
	data.NavigationContext
	Cards []DashboardCard
}
type DashboardCard struct {
	ID        string // "images", "albums"
	Title     string
	IconTitle string
	Icon      string
	URL       string
	Stats     []DashboardStat
}

type DashboardStat struct {
	Label string
	Value string
}
