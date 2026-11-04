package ocr

import "time"

// now gives the time. Tests replace it.
var now = func() time.Time { return time.Now().UTC() }
