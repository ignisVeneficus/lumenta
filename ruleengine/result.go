package ruleengine

import (
	"strconv"
	"time"
)

type TriState string

const (
	EvalResultTrue   TriState = "true"
	EvalResultFalse  TriState = "false"
	EvalResultUnknow TriState = "unknow"
)

func (ts TriState) Bool() bool {
	return ts == EvalResultTrue
}

type RuleEvaluation string

const (
	EvaluationFilesystem RuleEvaluation = "path_filter"
	EvaluationPanorama   RuleEvaluation = "panorama"
	EvaluationACL        RuleEvaluation = "acl"
)

var AllRuleEvaluation = []RuleEvaluation{
	EvaluationFilesystem,
	EvaluationACL,
	EvaluationPanorama,
}

type RuleParam struct {
	Name  string   `json:"name"`
	Value []string `json:"value"`
}

type RuleResult struct {
	Name   string      `json:"name"`
	Type   string      `json:"rule-type"`
	Op     string      `json:"rule-op"`
	Params []RuleParam `json:"rule-params"`
	Actual []RuleParam `json:"actual-values"`
	Result TriState    `json:"result"`
}

func CreateRuleParamString(name, value string) RuleParam {
	return RuleParam{
		Name:  name,
		Value: []string{value},
	}
}
func CreateRuleParamEmpty(name string) RuleParam {
	return RuleParam{
		Name: name,
	}
}
func CreateRuleParamStrings(name string, value []string) RuleParam {
	return RuleParam{
		Name:  name,
		Value: value,
	}
}
func CreateRuleParamInt(name string, value int) RuleParam {
	return RuleParam{
		Name:  name,
		Value: []string{strconv.Itoa(value)},
	}
}
func CreateRuleParamInts(name string, value []uint64) RuleParam {
	strValue := make([]string, len(value))
	for _, v := range value {
		strValue = append(strValue, strconv.Itoa(int(v)))
	}
	return RuleParam{
		Name:  name,
		Value: strValue,
	}
}
func CreateRuleParamFloat64(name string, value float64) RuleParam {
	return RuleParam{
		Name:  name,
		Value: []string{strconv.FormatFloat(value, 'f', -1, 64)},
	}
}
func CreateRuleParamDate(name string, value time.Time) RuleParam {
	return RuleParam{
		Name:  name,
		Value: []string{value.Format("2006.01.02 15:04:05")},
	}
}

type GroupRuleResult struct {
	Name        string       `json:"name"`
	Op          RuleGroupOp  `json:"op"`
	Params      []RuleParam  `json:"params"`
	RuleResults []RuleResult `json:"rule_results"`
	Result      bool         `json:"result"`
}

type RuleResults map[RuleEvaluation][]GroupRuleResult

func (rr *RuleResults) AddResult(name RuleEvaluation, result GroupRuleResult) {
	if *rr == nil {
		*rr = make(RuleResults)
	}
	(*rr)[name] = append((*rr)[name], result)
}
