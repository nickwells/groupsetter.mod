package groupsetter

import (
	"fmt"
	"strings"

	"github.com/nickwells/param.mod/v7/psetter"
	"github.com/nickwells/param.mod/v7/ptypes"
)

// List allows you to give a parameter that can be used to set a
// list (a slice) of groups of parameters. Note that this requires careful
// use; this setter must be created and then the internal parameters that are
// added to this must have setters that take the fields from the InterimValue
// as their value.
type List[T any] struct {
	*GroupParams[T]
	psetter.ValueReqMandatory

	// Value must be set, the program will panic if not. This is the slice of
	// vaules that the setter is setting.
	Value *[]T
}

// NewList constructs a List and returns it. Lists must be created through
// this function - the CheckSetter method will panic if not.
func NewList[T any](
	v *[]T, optFuncs ...ptypes.OptFunc[GroupParams[T]],
) *List[T] {
	s := &List[T]{
		GroupParams: NewGroupParams(optFuncs...),
		Value:       v,
	}

	return s
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
	if err := s.PopulateInterimValue(paramName, paramVal); err != nil {
		return err
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

	s.WriteByPosAllowedValues(&avStr)

	s.WriteByNameAllowedValues(&avStr)

	return avStr.String()
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

	s.CheckGroupParams(name)
}

// ValDescribe returns a name describing the values allowed
func (s List[T]) ValDescribe() string {
	return "group-of-values"
}
