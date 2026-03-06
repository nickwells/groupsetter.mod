package groupsetter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v7/param"
	"github.com/nickwells/param.mod/v7/phelp"
	"github.com/nickwells/param.mod/v7/psetter"
	"github.com/nickwells/param.mod/v7/ptypes"
)

// InterimValueInitFunc is the type of a function that will set the initial
// value of an InterimValue in a groupsetter
type InterimValueInitFunc[T any] func(*T) error

// SetInterimValueInitFunc returns an option function that will set the
// function used to set the initial value of the interim value in a
// GroupParams. The function must not be nil, an error is returned if it is.
func SetInterimValueInitFunc[T any](
	f InterimValueInitFunc[T],
) ptypes.OptFunc[GroupParams[T]] {
	return func(s *GroupParams[T]) error {
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

// GroupParams provides features that are common across many of the
// Setters in this module.
type GroupParams[T any] struct {
	// pSet is the parameter set which is used to
	//   - check that added parameters and final checks are valid.
	//   - generate the text of the allowed values
	//
	// The parameter set used to actually parse the per-instance parameter is
	// constructed afresh from the ...Cache values below for each invocation
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

	// ivInitialiser is an optional function that can set the initial value
	// of the InterimValue. It will be called before the group parameters are
	// parsed.
	ivInitialiser InterimValueInitFunc[T]

	// InterimVal is the value that is set each time the parameter is
	// given. Following a successful parsing of the group's internal
	// parameters this value will have been successfully populated and will
	// be added to the Value slice. This should be the target of the
	// parameters added by the AddParam method.
	InterimVal T

	// The StrListSeparator allows you to override the default separator
	// between the parameters in the group.
	psetter.StrListSeparator
}

// NewGroupParams initialises and returns a properly constructed
// GroupParams. The group parameter separator is set by default to ";" but
// can be changed later before parameter parsing.
func NewGroupParams[T any](
	optFuncs ...ptypes.OptFunc[GroupParams[T]],
) *GroupParams[T] {
	bs := &GroupParams[T]{
		pSet: param.NewSet(
			phelp.NoHelp{},
			param.SetParamPrefixes()),
		avalShownAlready: make(ptypes.AValCache),
		StrListSeparator: psetter.StrListSeparator{Sep: ";"},
	}

	for _, o := range optFuncs {
		err := o(bs)
		if err != nil {
			panic(
				fmt.Sprintf("could not create a new GroupParams: %v", err))
		}
	}

	return bs
}

// AddByNameParam adds a new ByName parameter to the internal parameter set.
func (s *GroupParams[T]) AddByNameParam(
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
func (s *GroupParams[T]) AddByPosParam(
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
func (s *GroupParams[T]) AddFinalCheck(f param.FinalCheckFunc) {
	s.finalCheckCache = append(s.finalCheckCache, f)
	s.pSet.AddFinalCheck(f)
}

// initLocalPSet creates a new local instance of the parameter set so that
// the latest instance of the parameter can be parsed without complaining
// that parsing has already taken place.
func (s *GroupParams[T]) initLocalPSet() *param.PSet {
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

// resetInterimValue sets the InterimVal to its initial settings. It will
// return a non-nil error if the ivInitialiser is set and that function
// returns a non-nil error.
func (s *GroupParams[T]) resetInterimValue() error {
	var v T
	if s.ivInitialiser != nil {
		if err := s.ivInitialiser(&v); err != nil {
			return err
		}
	}

	s.InterimVal = v

	return nil
}

// PopulateInterimValue will reset the interim value, construct a local
// parameter set (ready for parsing) and split the paramVal according to the
// StrListSeparator. It will then parse the list of values split from the
// paramVal, thereby populating the interim value. Any problems with
// resetting the interim value or with the parsing of the split ParamVal will
// cause a non-nil error to be returned. If there are no problems then a nil
// error is returned and the GroupParams.InterimVal can be assumed to be
// successfully populated.
func (s *GroupParams[T]) PopulateInterimValue(
	paramName, paramVal string,
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

	return nil
}

// CheckGroupParams panics if the setter has not been properly created.
func (s GroupParams[T]) CheckGroupParams(name string) {
	// Check the GroupParams has been properly constructed
	if s.pSet == nil {
		panic(psetter.BadSetterMessage(
			name,
			fmt.Sprintf("%T", s),
			"use the constructor"))
	}
}
