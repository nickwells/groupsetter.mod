package groupsetter

import (
	"fmt"
	"strings"

	"github.com/nickwells/param.mod/v7/param"
	"github.com/nickwells/param.mod/v7/phelputils"
)

// getByNameParams returns all the allowed ByName parameters for this group
func (s GroupParams[T]) getByNameParams() []*param.ByName {
	params := []*param.ByName{}

	for _, g := range s.pSet.GetGroups() {
		params = append(params, g.Params()...)
	}

	return params
}

// getByPosParams returns all the allowed ByPos parameters for this group
func (s GroupParams[T]) getByPosParams() []*param.ByPos {
	params := []*param.ByPos{}

	for i := range s.pSet.CountByPosParams() {
		bpp, err := s.pSet.GetParamByPos(i)
		if err != nil {
			break
		}

		params = append(params, bpp)
	}

	return params
}

// WriteByNameAllowedValues writes the ByName parameters (if any) and their
// allowed values to the supplied string Builder.
func (s GroupParams[T]) WriteByNameAllowedValues(avStr *strings.Builder) {
	params := s.getByNameParams()
	if len(params) == 0 {
		return
	}

	avStr.WriteString("the following ")

	if len(params) == 1 {
		avStr.WriteString("parameter may appear ")
	} else {
		fmt.Fprintf(avStr,
			"%d parameters may appear in any order ",
			len(params))
	}

	avStr.WriteString("but you must give the name and '='" +
		" (if a following value is required)\n\n")

	pSep := ""
	for _, p := range params {
		avStr.WriteString(pSep)
		pSep = "\n\n;\n\n"

		avStr.WriteString(phelputils.ParamSummary(*p) + "\n\n")
		avStr.WriteString(p.Description() + "\n\n")
		avStr.WriteString("Allowed Values:" + "\n")

		avSep := ""

		avParts := phelputils.AllowedValueParts(
			s.avalShownAlready,
			p.Name(),
			p.Setter())
		for _, avp := range avParts {
			avStr.WriteString(avSep)
			avSep = "\n"

			avStr.WriteString(avp)
		}
	}
}

// WriteByPosAllowedValues writes the ByPos parameters (if any) and their
// allowed values to the supplied string Builder.
func (s GroupParams[T]) WriteByPosAllowedValues(avStr *strings.Builder) {
	params := s.getByPosParams()
	if len(params) == 0 {
		return
	}

	avStr.WriteString("the following ")

	if len(params) == 1 {
		avStr.WriteString("parameter must appear ")
	} else {
		fmt.Fprintf(avStr,
			"%d parameters must appear in the order shown below ",
			len(params))
	}

	avStr.WriteString("at the start (without the name and '=')\n\n")

	pSep := ""
	for _, p := range params {
		avStr.WriteString(pSep)
		pSep = "\n\n;\n\n"

		avStr.WriteString(p.Name() + "\n\n" + p.Description() + "\n\n")
		avStr.WriteString("Allowed Values:" + "\n")

		avSep := ""

		avParts := phelputils.AllowedValueParts(
			s.avalShownAlready, p.Name(), p.Setter())
		for _, avp := range avParts {
			avStr.WriteString(avSep)
			avSep = "\n\n"

			avStr.WriteString(avp)
		}
	}

	avStr.WriteString("\n\n")
}
