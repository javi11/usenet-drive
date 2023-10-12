package connectionpool

import (
	"strings"

	"github.com/chrisfarms/nntp"
)

var retirableErrors = []uint{
	441,
}

func IsRetryable(err error) bool {
	if strings.Contains(err.Error(), "broken pipe") {
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
