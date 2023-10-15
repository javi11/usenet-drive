package connectionpool

import (
	"errors"
	"syscall"

	"github.com/chrisfarms/nntp"
)

var retirableErrors = []uint{
	441,
}

func IsRetryable(err error) bool {
	if errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	if _, ok := err.(nntp.ProtocolError); ok {
		return true
	}

	if e, ok := err.(nntp.Error); ok {
		for _, r := range retirableErrors {
			if e.Code == r {
				return true
			}
		}
	}

	return false
}
