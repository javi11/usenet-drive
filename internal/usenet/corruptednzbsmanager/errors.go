package corruptednzbsmanager

import (
	"fmt"
)

type ErrCorruptedNzb struct {
	Err     error
	Segment *NotFoundSegment
}

type NotFoundSegment struct {
	Id     string
	Number int64
}

func NewCorruptedNzbError(err error, segment *NotFoundSegment) *ErrCorruptedNzb {
	return &ErrCorruptedNzb{
		Segment: segment,
		Err:     err,
	}
}

func (cn *ErrCorruptedNzb) Error() string {
	return fmt.Sprintf("corrupted nzb: %v", cn.Err)
}

func (cn *ErrCorruptedNzb) Unwrap() error {
	return cn.Err
}

func IsCorruptedNzbErr(target error) bool {
	_, ok := target.(*ErrCorruptedNzb)
	return ok
}
