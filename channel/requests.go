package channel

import (
	"fmt"
	"time"
)

type playRequest struct{ reqTime time.Time }

func (p playRequest) String() string {
	return fmt.Sprintf("Play request @ %s", p.reqTime.Format(time.DateTime))
}

type stopRequest struct{ reqTime time.Time }

func (s stopRequest) String() string {
	return fmt.Sprintf("Stop request @ %s", s.reqTime.Format(time.DateTime))
}

type skipRequest struct{ reqTime time.Time }

func (s skipRequest) String() string {
	return fmt.Sprintf("Skip request @ %s", s.reqTime.Format(time.DateTime))
}
