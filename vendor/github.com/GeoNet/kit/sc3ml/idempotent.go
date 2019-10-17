package sc3ml

import (
	"sync"
	"time"
)

// IdpQuake can be used to implement an idempotent receiver for Quakes.
// Thread safe.
type IdpQuake struct {
	idp map[string]Quake
	mu  sync.RWMutex
}

// Seen returns true if the Quake q has been previously
// seen via Add().  False otherwise.
// Quakes older than 60 minutes are evicted from i before checking for q.
func (i *IdpQuake) Seen(q Quake) bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.idp == nil {
		i.idp = make(map[string]Quake)
	}

	c := time.Now().UTC().Add(time.Duration(-60 * time.Minute))

	for _, e := range i.idp {
		if e.Time.Before(c) {
			delete(i.idp, e.PublicID)
		}
	}

	_, b := i.idp[q.PublicID]

	return b
}

// Add adds Quake.
func (i *IdpQuake) Add(q Quake) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.idp[q.PublicID] = q
}
