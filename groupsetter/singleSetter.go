package groupsetter

import (
	"fmt"
	"strings"

	"github.com/nickwells/param.mod/v7/psetter"
	"github.com/nickwells/param.mod/v7/ptypes"
)

// Single allows you to give a parameter that takes a group of parameters
// together. It is strongly advised that this param.Setter is constructed
// using the NewSingle func as this will ensure that it is properly
// constructed.
type Single[T any] struct {
	*GroupParams[T]
	psetter.ValueReqMandatory

	// Value must be set, the program will panic if not. This is the vaule
	// that the setter is setting.
	Value *T
}

// NewSingle constructs a Single setter and returns it. It is strongly
// advised that this param.Setter is constructed using this func as this will
// ensure that it is properly constructed.
func NewSingle[T any](
	v *T, optFuncs ...ptypes.OptFunc[GroupParams[T]],
) *Single[T] {
	s := &Single[T]{
		GroupParams: NewGroupParams(optFuncs...),
		Value:       v,
	}

	return s
}

// SetWithVal (called when a value follows the parameter) populates the
// InterimValue using the individual Setters added to this Setter. If no
// problems are found the new (composite) value is written to the
// Single.Value.
//
// If any problems are found this will return a non-nil error and the Value
// is not updated.
func (s *Single[T]) SetWithVal(
	paramName string,
	paramVal string,
) error {
	if err := s.PopulateInterimValue(paramName, paramVal); err != nil {
		return err
	}

	*s.Value = s.InterimVal

	return nil
}

// AllowedValues returns a description of the allowed values. It includes the
// separator to be used
func (s Single[T]) AllowedValues() string {
	var avStr strings.Builder

	avStr.WriteString("a collection of parameters, ")
	fmt.Fprintf(&avStr, "separated by %q", s.GetSeparator())
	avStr.WriteString(", used to collectively set a new value.")
	avStr.WriteString(" The parameters allowed are:\n\n")

	s.WriteByPosAllowedValues(&avStr)

	s.WriteByNameAllowedValues(&avStr)

	return avStr.String()
}

// CurrentValue returns the current setting of the parameter value
func (s Single[T]) CurrentValue() string {
	return fmt.Sprintf("%v", *s.Value)
}

// CheckSetter panics if the setter has not been properly created - if the
// Value is nil or if it has not been constructed using the NewSingle function.
func (s Single[T]) CheckSetter(name string) {
	// Check the value is not nil
	if s.Value == nil {
		panic(psetter.NilValueMessage(name, fmt.Sprintf("%T", s)))
	}

	s.CheckGroupParams(name)
}
