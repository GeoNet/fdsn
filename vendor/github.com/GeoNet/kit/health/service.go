package health

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

// CheckPath is the baked in SOH endpoint path.
const CheckPath = "/soh"

// Service provides a mechanism to update a service SOH status.
type Service struct {
	mu sync.Mutex

	// status is used to indicate whether the service is running
	status bool
	// last stores the time of the last update.
	last time.Time

	// start stores when the service was started.
	start time.Time
	// aged is the time if no updates have happened indicates the service is no longer running.
	// Default zero value means no age check required.
	aged time.Duration
	// startup is the time after the start which the check is assumed to be successful.
	startup time.Duration
}

// New returns a health Service which provides running SOH capabilities.
func New(endpoint string, aged, startup time.Duration) *Service {
	service := &Service{
		aged:    aged,
		last:    time.Now(),
		start:   time.Now(),
		startup: startup,
	}

	router := http.NewServeMux()
	router.HandleFunc(CheckPath, service.handler)

	srv := &http.Server{
		Addr:              endpoint,
		Handler:           router,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println("error starting health check service", err)
		}
	}()

	return service
}

// state returns the current application state, this is likely to
// be expanded as new checks are added.
func (s *Service) state() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.status
}

func (s *Service) handler(w http.ResponseWriter, r *http.Request) {
	ok := s.state()
	switch {
	case time.Since(s.start) < s.startup:
		// Avoid terminating before initial check period
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("warn")); err != nil {
			log.Println("error writing response", err)
		}
	case ok && (s.aged == 0 || time.Since(s.last) < s.aged):
		// Service is OK and actively updating
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Println("error writing response", err)
		}
	default:
		// Service is not OK or has stopped updating
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("fail")); err != nil {
			log.Println("error writing response", err)
		}
	}
}

// Ok updates the Service to indicate the service is running as expected.
func (s *Service) Ok() {
	s.Update(true)
}

// Fail updates the Service to indicate the service is not running as expected.
func (s *Service) Fail() {
	s.Update(false)
}

// Update sets the Service to the given state, and stores the time since the last update.
func (s *Service) Update(status bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = status
	s.last = time.Now()
}

// Alive allows an application to perform a complex task while still sending hearbeats.
func (s *Service) Alive(ctx context.Context, heartbeat time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer cancel()

		ticker := time.NewTicker(heartbeat)
		defer ticker.Stop()

		s.Ok()

		for {
			select {
			case <-ticker.C:
				s.Ok()
			case <-ctx.Done():
				return
			}
		}
	}()

	return cancel
}

// Pause allows an application to stall for a set period of time while still sending hearbeats.
func (s *Service) Pause(ctx context.Context, deadline, heartbeat time.Duration) context.CancelFunc {
	ctx, cancel := context.WithTimeout(ctx, deadline)

	go func() {
		defer cancel()

		ticker := time.NewTicker(heartbeat)
		defer ticker.Stop()

		s.Ok()

		for {
			select {
			case <-ticker.C:
				s.Ok()
			case <-ctx.Done():
				return
			}
		}
	}()

	return cancel
}
