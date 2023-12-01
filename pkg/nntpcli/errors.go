package nntpcli

import (
	"errors"
	"io"
	"net"
	"net/textproto"
	"syscall"
)

var (
	ErrCapabilitiesUnpopulated = errors.New("capabilities unpopulated")
	ErrNoSuchCapability        = errors.New("no such capability")
)

const SegmentAlreadyExistsErrCode = 441
const ToManyConnectionsErrCode = 502

var retirableErrors = []int{
	SegmentAlreadyExistsErrCode,
	ToManyConnectionsErrCode,
}

func IsRetryableError(err error) bool {
	if errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var netErr net.Error
	if ok := errors.As(err, &netErr); ok {
		return true
	}

	var protocolErr textproto.ProtocolError
	if ok := errors.As(err, &protocolErr); ok {
		return true
	}

	var nntpErr *textproto.Error
	if ok := errors.As(err, &nntpErr); ok {
		for _, r := range retirableErrors {
			if nntpErr.Code == r {
				return true
			}
		}
	}

	return false
}
