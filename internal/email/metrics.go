package email

import "sync/atomic"

var (
	emailSendOKTotal   atomic.Uint64
	emailSendFailTotal atomic.Uint64
)

// RecordSend increments the email send counter for the provided outcome.
func RecordSend(outcome string) {
	if outcome == "ok" {
		emailSendOKTotal.Add(1)
		return
	}
	emailSendFailTotal.Add(1)
}

// SendTotals returns the in-process email send counters.
func SendTotals() (ok uint64, fail uint64) {
	return emailSendOKTotal.Load(), emailSendFailTotal.Load()
}
