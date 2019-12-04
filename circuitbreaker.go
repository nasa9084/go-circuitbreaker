package circuitbreaker

import (
	"sync"
	"time"
)

type timer interface {
	Stop() bool
}

type state int8

const (
	closed state = iota
	halfopened
	opened
)

func (s state) String() string {
	switch s {
	case closed:
		return "closed"
	case halfopened:
		return "halfopened"
	case opened:
		return "opened"
	default:
		panic("unknown state")
	}
}

type counter struct {
	mu      sync.RWMutex
	success uint64
	failure uint64
}

func (cnt *counter) ResetSuccess() {
	cnt.mu.Lock()
	defer cnt.mu.Unlock()
	cnt.success = 0
}

func (cnt *counter) Success() uint64 {
	cnt.mu.Lock()
	defer cnt.mu.Unlock()
	cnt.success++
	return cnt.success
}

func (cnt *counter) ResetFail() {
	cnt.mu.Lock()
	defer cnt.mu.Unlock()
	cnt.failure = 0
}

func (cnt *counter) Fail() uint64 {
	cnt.mu.Lock()
	defer cnt.mu.Unlock()
	cnt.failure++
	return cnt.failure
}

// CircuitBreaker is a state machine which implements the Circuit Breaker pattern.
type CircuitBreaker struct {
	mu               sync.RWMutex
	counter          counter
	state            state
	successThreshold uint64
	failureThreshold uint64
	timeoutDuration  time.Duration
	timer            timer
}

// Default configuration values
const (
	DefaultSuccessThreshold = 3
	DefaultFailureThreshold = 3
	DefaultTimeoutDuration  = 30 * time.Second
)

// New returns a new Circuit Breaker which is configured with given options.
func New(options ...Option) *CircuitBreaker {
	cb := &CircuitBreaker{
		successThreshold: DefaultSuccessThreshold,
		failureThreshold: DefaultFailureThreshold,
		timeoutDuration:  DefaultTimeoutDuration,
	}
	for _, opt := range options {
		opt(cb)
	}
	return cb
}

func (cb *CircuitBreaker) IsAvail() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == closed || cb.state == halfopened
}

// afterFunc is just time.AfterFunc, for testing
var afterFunc func(time.Duration, func()) timer = func(d time.Duration, fn func()) timer { return time.AfterFunc(d, fn) }

// Fail sends failure signal to CircuitBreaker.
func (cb *CircuitBreaker) Fail() {
	switch cb.state {
	case opened:
		return
	case halfopened:
		cb.mu.Lock()
		cb.state = opened
		cb.timer = afterFunc(cb.timeoutDuration, cb.timeout)
		cb.mu.Unlock()
	case closed:
		if fail := cb.counter.Fail(); fail >= cb.failureThreshold {
			cb.mu.Lock()
			cb.state = opened
			cb.timer = afterFunc(cb.timeoutDuration, cb.timeout)
			cb.mu.Unlock()
		}
	}
}

// Success sends success signal to CircuitBreaker.
func (cb *CircuitBreaker) Success() {
	switch cb.state {
	case opened, closed:
		return
	case halfopened:
		if success := cb.counter.Success(); success >= cb.successThreshold {
			cb.mu.Lock()
			cb.counter.ResetFail()
			cb.state = closed
			cb.mu.Unlock()
		}
	}
}

func (cb *CircuitBreaker) timeout() {
	cb.mu.Lock()
	cb.counter.ResetSuccess()
	cb.state = halfopened
	cb.timer = nil
	cb.mu.Unlock()
}

// Reset CircuitBreaker state to initial state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.timer != nil {
		cb.timer.Stop()
		cb.timer = nil
	}
	cb.counter.ResetSuccess()
	cb.counter.ResetFail()
	cb.state = closed
}

// Option is an functional option for CircuitBreaker object.
type Option func(cb *CircuitBreaker)

// WithSuccessThreshold allows you to configure success threshold to CircuitBreaker.
func WithSuccessThreshold(threshold uint64) Option {
	return func(cb *CircuitBreaker) {
		cb.successThreshold = threshold
	}
}

// WithFailureThreshold allows you to configure error threshold to CircuitBreaker.
func WithFailureThreshold(threshold uint64) Option {
	return func(cb *CircuitBreaker) {
		cb.failureThreshold = threshold
	}
}

// WithTimeoutDuration allows you to configure the timeout duration for state transition from opened to half-opened.
func WithTimeoutDuration(d time.Duration) Option {
	return func(cb *CircuitBreaker) {
		cb.timeoutDuration = d
	}
}
