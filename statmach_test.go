package statmach

import (
	"errors"
	"testing"
)

const (
	TRIGGER1 = "trigger1"
	TRIGGER2 = "trigger2"
	TRIGGER3 = "trigger3"
	TRIGGER4 = "trigger4"
	TRIGGER5 = "trigger5"
)

func failIfExpectedStateIsNot(sm *StateMachine, expectedState string, t *testing.T) {
	if currStateName := sm.GetCurrentState().GetStateName(); currStateName != expectedState {
		t.Errorf("expected current state is %s, got %s", expectedState, currStateName)
	}
}

func failIfErrIsNotNil(err *error, t *testing.T, format string, args ...interface{}) {
	if err != nil {
		t.Errorf(format, args...)
	}
}

func failIfErrIsNil(err *error, t *testing.T, format string, args ...interface{}) {
	if err == nil {
		t.Errorf(format, args...)
	}
}

func errorFlowHandler(fn func(errCh chan error, doneCh chan interface{}), errHandler func(err error)) {
	errCh := make(chan error)
	doneCh := make(chan interface{})

	go fn(errCh, doneCh)

	for {
		select {
		case err := <-errCh:
			errHandler(err)
		case <-doneCh:
			close(errCh)
			close(doneCh)
			return
		}
	}
}

func TestBasicTransition(t *testing.T) {
	sm := New("src")
	sc := sm.Configure("src")
	sc.Permit(TRIGGER1, "dst")
	sc = sm.Configure("dst")
	sc.Permit(TRIGGER2, "dst1")
	sm.Fire(TRIGGER1)
	sm.Fire(TRIGGER2)
	failIfExpectedStateIsNot(sm, "dst1", t)
}

func TestConditionalTransition(t *testing.T) {
	sm := New("src")
	sc := sm.Configure("src")
	sc.PermitIf(TRIGGER1, "dst", func(...interface{}) bool {
		return false
	})
	failIfExpectedStateIsNot(sm, "src", t)
}

func TestBasicHierarchicalTransition(t *testing.T) {
	errorFlowHandler(func(errCh chan error, doneCh chan interface{}) {
		sm := New("src")
		// configure src
		sc := sm.Configure("src")
		errCh <- sc.Permit(TRIGGER1, "dst1")
		errCh <- sc.Permit(TRIGGER2, "dst2")

		// configure dst2
		sc = sm.Configure("dst2")
		errCh <- sc.SubstateOf("src")

		_, err := sm.Fire(TRIGGER2)
		errCh <- err
		_, err = sm.Fire(TRIGGER1)
		errCh <- err
		if sm.GetCurrentState().GetStateName() != "dst1" {
			errCh <- errors.New("expected state should be dst1")
		}
		doneCh <- nil
	}, func(err error) {
		if err != nil {
			t.Error(err)
		}
	})
}

func TestStateShouldNotHaveOneMoreSameTrigger(t *testing.T) {
	sm := New("src")
	sc := sm.Configure("src")
	sc.Permit(TRIGGER1, "dst")

	err := sc.Permit(TRIGGER1, "dst1")
	failIfErrIsNil(&err, t, "src should have one or less %s", TRIGGER1)
	err = sc.PermitIf(TRIGGER1, "dst2", func(...interface{}) bool {
		return true
	})
	failIfErrIsNil(&err, t, "src should have one or less %s", TRIGGER1)
}

func TestStatesCannotBeSubstatesOfEachOther(t *testing.T) {
	sm := New("src")
	sc := sm.Configure("src")
	sc.SubstateOf("dst")

	sc = sm.Configure("dst")
	err := sc.SubstateOf("src")
	if err == nil {
		t.Error("StatesCannotBeSubstatesOfEachOther")
	}
}
