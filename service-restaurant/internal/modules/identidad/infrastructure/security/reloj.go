package security

import "time"

// SystemClock implementa outbound.Clock con la hora real (UTC).
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
