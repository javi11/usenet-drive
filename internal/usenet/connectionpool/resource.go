//go:generate mockgen -source=./resource.go -destination=./resource_mock.go -package=connectionpool Resource

package connectionpool

import (
	"time"

	"github.com/javi11/usenet-drive/pkg/nntpcli"
)

type Resource interface {
	CreationTime() time.Time
	Destroy()
	Hijack()
	IdleDuration() time.Duration
	LastUsedNanotime() int64
	Release()
	ReleaseUnused()
	Value() nntpcli.Connection
}
