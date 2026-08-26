package scan

import "testing"

func TestTrackerArrivalThenStable(t *testing.T) {
	tr := NewTracker(2)
	s := tr.Update([]string{"a"})
	if s["a"] != Arrived {
		t.Fatal("first sight must be Arrived")
	}
	s = tr.Update([]string{"a"})
	if s["a"] != Arrived {
		t.Fatal("second cycle still lit")
	}
	s = tr.Update([]string{"a"})
	if s["a"] != Stable {
		t.Fatal("third cycle must settle")
	}
}

func TestTrackerDepartingThenGone(t *testing.T) {
	tr := NewTracker(1)
	tr.Update([]string{"a"})
	tr.Update([]string{"a"}) // settled
	s := tr.Update(nil)
	if s["a"] != Departing {
		t.Fatalf("missing row must be Departing, got %v", s["a"])
	}
	s = tr.Update(nil)
	if _, ok := s["a"]; ok {
		t.Fatal("departing row must be removed after one cycle")
	}
}

func TestTrackerComebackCancelsDeparture(t *testing.T) {
	tr := NewTracker(1)
	tr.Update([]string{"a"})
	tr.Update(nil) // departing
	s := tr.Update([]string{"a"})
	if s["a"] == Departing {
		t.Fatal("comeback must cancel departure")
	}
	if s["a"] == Departing {
		t.Fatal("still departing")
	}
}
