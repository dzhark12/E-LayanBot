package domain

import "time"

type Lampiran struct {
	ID         string
	Jenis      string
	MIME       string
	Ukuran     int64
	DibuatPada time.Time
}
