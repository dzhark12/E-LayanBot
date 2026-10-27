package domain

import "time"

type SesiBot struct {
	JID        string
	Flow       string
	Step       string
	Data       []byte
	DiubahPada time.Time
}
