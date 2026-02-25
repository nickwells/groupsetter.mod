package groupsetter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v7/param"
	"github.com/nickwells/param.mod/v7/phelp"
	"github.com/nickwells/param.mod/v7/phelputils"
	"github.com/nickwells/param.mod/v7/psetter"
	"github.com/nickwells/param.mod/v7/ptypes"
)

// InterimValueInitFunc is the type of a function that will set the initial
// value of an InterimValue in a List
type InterimValueInitFunc[T any] func(*T) error

// ListOFSetInterimValueInitFunc returns an option function that will set the
// function used to set the initial value of the interim value in a
// List. The function must not be nil, an error is returned if it
// is.
func ListOFSetInterimValueInitFunc[T any](
	f InterimValueInitFunc[T],
) ptypes.OptFunc[List[T]] {
	return func(s *List[T]) error {
		if f == nil {
			return errors.New("a nil InterimValueInitFunc is not allowed")
		}

		s.ivInitialiser = f

		return nil
	}
}

// byNameParamInitInfo holds the parameters passed to the AddByNameParam
// method
type byNameParamInitInfo struct {
	name string
	s    param.Setter
	desc string
	opts []param.ByNameOptFunc
}

// byPosParamInitInfo holds the parameters passed to the AddByNameParam
// method
type byPosParamInitInfo struct {
	name string
	s    param.Setter
	desc string
	opts []param.ByPosOptFunc
}

// List allows you to give a parameter that can be used to set a
// list (a slice) of groups of parameters. Note that this requires careful
// use; this setter must be created and then the internal parameters that are
// added to this must have setters that take the fields from the InterimValue
// as their value.
type List[T any] struct {
	psetter.ValueReqMandatory

	// pSet is the parameter set which is used to check that added parameters
	// and final checks are valid. The parameter set used to actually parse
	// the per-instance parameter is constructed afresh for each invocation
	// to avoid a panic due to repeat parsing
	pSet *param.PSet
	// The following caches are used to construct the per-instance parameter
	// set used to parse values
	byNameCache     []byNameParamInitInfo
	byPosCache      []byPosParamInitInfo
	finalCheckCache []param.FinalCheckFunc
	// avalShownAlready records whether or not an allowed value description
	// has been reported before
	avalShownAlready ptypes.AValCache

	// Value must be set, the program will panic if not. This is the slice of
	// vaules that the setter is setting.
	Value *[]T
	// InterimVal is the value that is set each time the parameter is
	// given. Following a successful parsing of the group's internal
	// parameters this value will have been successfully populated and will
	// be added to the Value slice. This should be the target of the
	// parameters added by the AddParam method.
	InterimVal T
	// ivInitialiser is an optional function that can set the initial value
	// of the InterimValue. It will be called before the group parameters are
	// parsed.
	ivInitialiser InterimValueInitFunc[T]
	// The StrListSeparator allows you to override the default separator
	// between the intra-group parameters.
	psetter.StrListSeparator
}

// NewList constructs a List and returns
// it. Lists must be created through this function - the
// CheckSetter method will panic if not.
func NewList[T any](
	v *[]T, opt ...ptypes.OptFunc[List[T]],
) *List[T] {
	s := &List[T]{
		Value: v,
		pSet: param.NewSet(
			phelp.NoHelp{},
			param.SetParamPrefixes()),
		StrListSeparator: psetter.StrListSeparator{Sep: ";"},
		avalShownAlready: make(ptypes.AValCache),
	}

	for _, o := range opt {
		err := o(s)
		if err != nil {
			panic(
				fmt.Sprintf("could not create a new List: %v", err))
		}
	}

	return s
}

// AddByNameParam adds a new ByName parameter to the internal parameter set.
func (s *List[T]) AddByNameParam(
	name string, setter param.Setter, desc string, opts ...param.ByNameOptFunc,
) *param.ByName {
	s.byNameCache = append(s.byNameCache, byNameParamInitInfo{
		name: name,
		s:    setter,
		desc: desc,
		opts: opts,
	})

	return s.pSet.Add(name, setter, desc, opts...)
}

// AddByPosParam adds a new ByPos parameter to the internal parameter set.
func (s *List[T]) AddByPosParam(
	name string, setter param.Setter, desc string, opts ...param.ByPosOptFunc,
) *param.ByPos {
	s.byPosCache = append(s.byPosCache, byPosParamInitInfo{
		name: name,
		s:    setter,
		desc: desc,
		opts: opts,
	})

	return s.pSet.AddByPos(name, setter, desc, opts...)
}

// AddFinalCheck adds a new FinalCheckFunc to the internal parameter set.
func (s *List[T]) AddFinalCheck(f param.FinalCheckFunc) {
	s.finalCheckCache = append(s.finalCheckCache, f)
	s.pSet.AddFinalCheck(f)
}

// resetInterimValue sets the InterimVal to its initial settings. It will
// return a non-nil error if the ivInitialiser is set and that function
// returns a non-nil error.
func (s *List[T]) resetInterimValue() error {
	var v T
	if s.ivInitialiser != nil {
		if err := s.ivInitialiser(&v); err != nil {
			return err
		}
	}

	s.InterimVal = v

	return nil
}

// initLocalPSet creates a new local instance of the parameter set so that
// the latest instance of the parameter can be parsed without complaining
// that parsing has already taken place.
func (s *List[T]) initLocalPSet() *param.PSet {
	localPSet := param.NewSet(phelp.NoHelp{}, param.SetParamPrefixes())
	for _, bpInfo := range s.byPosCache {
		localPSet.AddByPos(bpInfo.name, bpInfo.s, bpInfo.desc, bpInfo.opts...)
	}

	for _, bnInfo := range s.byNameCache {
		localPSet.Add(bnInfo.name, bnInfo.s, bnInfo.desc, bnInfo.opts...)
	}

	for _, f := range s.finalCheckCache {
		localPSet.AddFinalCheck(f)
	}

	return localPSet
}

// SetWithVal (called when a value follows the parameter) splits the value on
// the separator into a slice of strings, initialises a new interim value (of
// type T), generates a new param.PSet and parses the slice of strings to
// populate the interim value. If no problems are found the new value is
// added to the List.Value (a slice of elements of type T).
//
// If any problems are found this will return a non-nil error and the Value
// is not updated.
func (s *List[T]) SetWithVal(
	paramName string,
	paramVal string,
) error {
	sep := s.GetSeparator()
	sv := strings.Split(paramVal, sep)

	if err := s.resetInterimValue(); err != nil {
		return err
	}

	localPSet := s.initLocalPSet()

	loc := location.New(fmt.Sprintf("%q parameter", paramName))
	localPSet.ParamParse(loc, sv)

	errMap := localPSet.Errors()
	if len(errMap) != 0 {
		var ew strings.Builder
		errMap.Report(&ew, paramName)

		return errors.New(ew.String())
	}

	*s.Value = append(*s.Value, s.InterimVal)

	return nil
}

// AllowedValues returns a description of the allowed values. It includes the
// separator to be used
func (s List[T]) AllowedValues() string {
	var avStr strings.Builder

	avStr.WriteString("a collection of parameters, ")
	fmt.Fprintf(&avStr, "separated by %q", s.GetSeparator())
	avStr.WriteString(", used to collectively set a new entry in a list.")
	avStr.WriteString(" The parameters allowed are:\n\n")

	s.writeByPosAllowedValues(&avStr)

	s.writeByNameAllowedValues(&avStr)

	return avStr.String()
}

// writeByNameAllowedValues writes the ByName parameters (if any) to the
// supplied string Builder.
func (s List[T]) writeByNameAllowedValues(avStr *strings.Builder) {
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

		avParts := phelputils.AllowedValueParts(s.avalShownAlready, p.Name(), p.Setter())
		for _, avp := range avParts {
			avStr.WriteString(avSep)
			avSep = "\n"

			avStr.WriteString(avp)
		}
	}
}

// writeByPosAllowedValues writes the ByPos parameters (if any) to the
// supplied string Builder.
func (s List[T]) writeByPosAllowedValues(avStr *strings.Builder) {
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

// getByNameParams returns all the allowed ByName parameters for this group
func (s List[T]) getByNameParams() []*param.ByName {
	params := []*param.ByName{}

	for _, g := range s.pSet.GetGroups() {
		params = append(params, g.Params()...)
	}

	return params
}

// getByPosParams returns all the allowed ByPos parameters for this group
func (s List[T]) getByPosParams() []*param.ByPos {
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

// CurrentValue returns the current setting of the parameter value
func (s List[T]) CurrentValue() string {
	var cv strings.Builder

	sep := ""

	for _, v := range *s.Value {
		cv.WriteString(sep)
		fmt.Fprintf(&cv, "%v", v)

		sep = "\n"
	}

	return cv.String()
}

// CheckSetter panics if the setter has not been properly created - if the
// Value is nil or if it has not been constructed using the NewList function.
func (s List[T]) CheckSetter(name string) {
	// Check the value is not nil
	if s.Value == nil {
		panic(psetter.NilValueMessage(
			name,
			fmt.Sprintf("%T", s)))
	}

	if s.pSet == nil {
		panic(psetter.BadSetterMessage(
			name,
			fmt.Sprintf("%T", s),
			"use the NewList function"))
	}
}

// ValDescribe returns a name describing the values allowed
func (s List[T]) ValDescribe() string {
	return "group-of-values"
}
