// Package id generates identifiers for the application.
//
// UUID v7 is used because it is time-ordered: the high bits carry a millisecond
// timestamp, so generated IDs sort by creation time. That keeps primary-key
// indexes dense (unlike UUID v4, which scatters inserts across the whole index)
// and makes range scans on the ID column meaningful.
package id

import (
	"github.com/google/uuid"
)

// NewV7 returns a new UUID v7 in its canonical 36-character string form.
//
// The value is generated in the application, never by the database, so the caller
// knows the ID before the INSERT and can log or enqueue it beforehand.
func NewV7() string {
	id, err := uuid.NewV7()
	if err != nil {
		// uuid.NewV7 only fails if the system random source fails, which is
		// unrecoverable. Panicking is correct: the process cannot safely
		// continue without unique identifiers.
		panic("id: generating uuid v7: " + err.Error())
	}

	return id.String()
}

// Parse validates a UUID string and returns it in canonical form.
//
// Use it at the boundary where an untrusted string enters the system, so that
// malformed identifiers are rejected before they reach the persistence layer.
func Parse(s string) (string, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return "", err
	}

	return parsed.String(), nil
}
