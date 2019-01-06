package bus

import (
	"fmt"
	"strings"
)

type Errors struct {
	Next *Errors
	Err  error
}

func (errs *Errors) Append(err error) *Errors {
	if errs == nil {
		return &Errors{Err: err}
	}
	if errs.Next == nil {
		errs.Next = &Errors{Err: err}
		return errs
	}
	errs.Next.Append(err)
	return errs
}
func (errs *Errors) AppendMsg(f string, v ...interface{}) *Errors {
	err := fmt.Errorf(f, v...)
	return errs.Append(err)
}
func (errs *Errors) Error() string {
	if errs == nil {
		return ""
	}
	b := &strings.Builder{}
	errs.writeTo(b)
	return b.String()
}
func (errs *Errors) writeTo(b *strings.Builder) {
	if errs == nil {
		return
	}
	if errs.Err != nil {
		b.WriteString(errs.Err.Error())
		b.WriteRune('\n')
	}
	if errs.Next != nil {
		errs.Next.writeTo(b)
	}
}
