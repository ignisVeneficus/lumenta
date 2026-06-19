package admin

import (
	"time"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	"github.com/ignisVeneficus/lumenta/tpl/data"
)

type SyncRunsPageContext struct {
	data.NavigationContext
	SyncRuns []SyncRunData
	Paging   data.Paging
}

type SyncRunData struct {
	dbo.SyncRun
}

func (sd *SyncRunData) RoutesSyncRunID() routes.SyncRunID {
	return routes.SyncRunID(*sd.ID)
}

func (sd *SyncRunData) Duration() *time.Duration {
	return calcDuration(&sd.StartedAt, sd.FinishedAt)
}

func calcDuration(start, end *time.Time) *time.Duration {
	if start == nil {
		return nil
	}

	var d time.Duration

	if end != nil {
		d = end.Sub(*start)
	} else {
		d = time.Since(*start)
	}
	return &d

}

type SyncFilesPageContext struct {
	data.NavigationContext
	SyncFiles   []SyncFileData
	Paging      data.Paging
	HasSearch   bool
	SearchField string
	HasFilter   bool
	Filter      []string
	FilterData  data.DropDown

	HasFileButton bool
}

type SyncFileData struct {
	dbo.SyncFile
	RuleResult ruleengine.RuleResults
}

func (sd *SyncFileData) RoutesSyncRunID() routes.SyncRunID {
	return routes.SyncRunID(sd.SyncID)
}
func (sd *SyncFileData) RoutesSyncFileID() routes.SyncFileID {
	return routes.SyncFileID(*sd.ID)
}

func (sf SyncFileData) ResultOrder() []ruleengine.RuleEvaluation {
	ret := ruleengine.AllRuleEvaluation
	orderSet := make(map[ruleengine.RuleEvaluation]struct{}, len(ret))
	for _, k := range ret {
		orderSet[k] = struct{}{}
	}

	for k := range sf.RuleResult {
		if _, ok := orderSet[k]; !ok {
			ret = append(ret, k)
		}
	}

	return ret
}

func (sf SyncFileData) Age() *time.Duration {
	return tpl.CalcDuration(&sf.CreatedAt, nil)
}
func (sf SyncFileData) FullPathText() string {
	return tpl.CreateSpacePath(sf.PathFull())
}
func (sf SyncFileData) DirtyReasonText() string {
	if sf.DirtyReason == nil {
		return ""
	}
	return *sf.DirtyReason
}

type SyncFilePageContext struct {
	data.NavigationContext
	File SyncFileData
}
