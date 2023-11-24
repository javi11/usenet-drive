package nntpcli

import (
	"errors"
	"fmt"
	"net"
	"syscall"
)

const SegmentAlreadyExistsErrCode = 441
const ToManyConnectionsErrCode = 502

// A ProtocolError represents responses from an NNTP server
// that seem incorrect for NNTP.
type ProtocolError string

func (p ProtocolError) Error() string {
	return string(p)
}

// An Error represents an error response from an NNTP server.
type NntpError struct {
	Code uint
	Msg  string
}

func (e NntpError) Error() string {
	return fmt.Sprintf("%03d %s", e.Code, e.Msg)
}

var retirableErrors = []uint{
	SegmentAlreadyExistsErrCode,
	ToManyConnectionsErrCode,
}

func IsRetryableError(err error) bool {
	if errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	var netErr net.Error
	if ok := errors.As(err, &netErr); ok {
		return true
	}

	var protocolErr ProtocolError
	if ok := errors.As(err, &protocolErr); ok {
		return true
	}

	var nntpErr NntpError
	if ok := errors.As(err, &nntpErr); ok {
		for _, r := range retirableErrors {
			if nntpErr.Code == r {
				return true
			}
		}
	}

	return false
}
