package nntpcli

import (
	"errors"
	"fmt"
	"net"
	"syscall"
)

const SegmentAlreadyExistsErrCode = 441

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
}

func IsRetryableError(err error) bool {
	if errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	if _, ok := err.(net.Error); ok {
		return true
	}

	if _, ok := err.(ProtocolError); ok {
		return true
	}

	if e, ok := err.(NntpError); ok {
		for _, r := range retirableErrors {
			if e.Code == r {
				return true
			}
		}
	}

	return false
}
