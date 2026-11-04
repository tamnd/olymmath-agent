package lane

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// QuotaError is a spent allowance, with the time when the provider said that it comes back.
type QuotaError struct {
	// Lane is the lane that ran out.
	Lane string
	// Message is the line in which the provider said so.
	Message string
	// Until is the time that the provider gave, or zero when it gave none.
	Until time.Time
}

func (e *QuotaError) Error() string {
	s := fmt.Sprintf("%s: %s", e.Lane, ErrExhausted)
	if e.Message != "" {
		s += ": " + e.Message
	}
	return s
}

// Unwrap lets errors.Is find ErrExhausted.
func (e *QuotaError) Unwrap() error { return ErrExhausted }

// DefaultRest is how long a lane rests after a spent allowance when the provider gives no time.
const DefaultRest = time.Hour

// Rest is how long to stay off the lane after now.
func (e *QuotaError) Rest(now time.Time) time.Duration {
	if e.Until.IsZero() {
		return DefaultRest
	}
	return max(0, e.Until.Sub(now))
}

// quotaPhrases are the phrases with which the clients say that an allowance is spent. They are whole phrases, because the word "limit" alone is also in an error about the length of the context, which is another failure.
var quotaPhrases = []string{
	"hit your usage limit",
	"usage limit reached",
	"reached your usage limit",
	"freeusagelimiterror",
	"rate limit exceeded",
	"quota exceeded",
	"exceeded your current quota",
	"insufficient_quota",
	"resource_exhausted",
	"credit balance is too low",
}

// tryAgainAt reads the time in a message such as "try again at Nov 9th, 2026 4:38 PM".
var tryAgainAt = regexp.MustCompile(`(?i)try again at ([A-Za-z]{3})[a-z]* (\d{1,2})(?:st|nd|rd|th)?,? (\d{4}) (\d{1,2}):(\d{2})\s*(AM|PM)`)

// Quota reads what a client said and gives a QuotaError when the allowance is spent, or nil.
func Quota(lane, said string) *QuotaError {
	low := strings.ToLower(strings.ReplaceAll(said, "’", "'"))
	line := ""
	for _, l := range strings.Split(low, "\n") {
		for _, p := range quotaPhrases {
			if line == "" && strings.Contains(l, p) {
				line = l
			}
		}
	}
	if line == "" {
		return nil
	}
	e := &QuotaError{Lane: lane, Message: oneLine(strings.TrimPrefix(strings.TrimSpace(line), "error: "))}
	if m := tryAgainAt.FindStringSubmatch(said); m != nil {
		s := fmt.Sprintf("%s %s %s %s:%s %s", strings.ToUpper(m[1][:1])+strings.ToLower(m[1][1:]), m[2], m[3], m[4], m[5], strings.ToUpper(m[6]))
		if t, err := time.ParseInLocation("Jan 2 2006 3:04 PM", s, time.Local); err == nil {
			e.Until = t
		}
	}
	return e
}

// oneLine keeps at most 200 bytes of a message, on one line.
func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
