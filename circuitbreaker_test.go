package circuitbreaker

import (
	"testing"
	"time"
)

type testTimer struct {
	d         time.Duration
	cur       time.Duration
	afterFunc func()
}

func testAfterFunc(d time.Duration, fn func()) timer {
	return &testTimer{
		d:         d,
		afterFunc: fn,
	}
}

func (t *testTimer) Stop() bool {
	t = &testTimer{}
	return true
}

func (t *testTimer) FastForward(d time.Duration) {
	t.cur += d
	if t.cur >= t.d {
		t.afterFunc()
		t.Stop()
	}
}

func init() {
	afterFunc = testAfterFunc
}

func assertState(t *testing.T, cb *CircuitBreaker, state state) {
	t.Helper()
	if cb.state != state {
		t.Errorf("unexpected state: %s != %s", cb.state, state)
		return
	}
	if state == closed || state == halfopened {
		if !cb.IsAvail() {
			t.Error("expected available but not")
			return
		}
	} else {
		if cb.IsAvail() {
			t.Error("expected unavailable but available")
			return
		}
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := New()
	assertState(t, cb, closed)

	cb.Fail()
	assertState(t, cb, closed)

	cb.Fail()
	assertState(t, cb, closed)

	cb.Fail() // 3 times failed
	assertState(t, cb, opened)

	timer := cb.timer.(*testTimer)
	timer.FastForward(29 * time.Second)
	assertState(t, cb, opened)

	timer.FastForward(1 * time.Second)
	assertState(t, cb, halfopened)
	if cb.counter.success != 0 {
		t.Error("cb.counter.success should be reset")
		return
	}

	cb.Fail() // re-fail
	assertState(t, cb, opened)

	timer = cb.timer.(*testTimer)
	timer.FastForward(30 * time.Second)
	assertState(t, cb, halfopened)

	cb.Success()
	cb.Success()
	// not closed yet
	assertState(t, cb, halfopened)

	cb.Success()
	assertState(t, cb, closed)

	if cb.timer != nil {
		t.Error("circuit breaker timer must be nil")
		return
	}
	if cb.counter.failure != 0 {
		t.Error("cb.counter.failure should be reset")
		return
	}
}
