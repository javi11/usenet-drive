package connectionpool

import "github.com/chrisfarms/nntp"

var retirableErrors = []uint{
	441,
}

func IsRetirableErr(err error) bool {
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
