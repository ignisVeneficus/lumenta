package data

import "github.com/ignisVeneficus/lumenta/internal/i18n"

type DropDown []DropDownData

func (dd *DropDown) Add(id, display, pill string) {
	if *dd == nil {
		*dd = make(DropDown, 0)
	}
	*dd = append(*dd, CreateDropDownData(id, display, pill))
}

func (dd *DropDown) AddT(id, display, pill, lang string, i18n *i18n.Service) {
	if *dd == nil {
		*dd = make(DropDown, 0)
	}
	*dd = append(*dd, CreateDropDownData(id, i18n.T(lang, display, nil), i18n.T(lang, pill, nil)))
}

type DropDownData struct {
	ID      string `json:"id"`
	Display string `json:"display"`
	Pill    string `json:"pill"`
}

func CreateDropDownData(id, display, pill string) DropDownData {
	return DropDownData{
		Pill:    pill,
		Display: display,
		ID:      id,
	}
}

func CreateDropDown[T ~string](vals []T, lang string, i18nKey string, i18n *i18n.Service) DropDown {
	ret := make(DropDown, len(vals))
	for i, v := range vals {
		ret[i] = CreateDropDownData(
			string(v),
			i18n.T(lang, i18nKey+"."+string(v)+".label", nil),
			i18n.T(lang, i18nKey+"."+string(v)+".short", nil),
		)
	}
	return ret

}
