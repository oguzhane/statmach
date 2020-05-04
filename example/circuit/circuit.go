package main

import (
	"errors"
	"log"
	"time"

	"github.com/OguzhanE/statmach"
)

// states
const (
	Closed   = "closed"
	Open     = "open"
	HalfOpen = "halfOpen"
)

// triggers
const (
	SuccessThresholdReached = "successThresholdReached"
	FailureThresholdReached = "failureThresholdReached"
	TimeoutTimerExpired     = "timeoutTimerExpired"
	OperationFailed         = "operationFailed"
	Try                     = "try"
)

const (
	successThreshold = 2
	failureThreshold = 2
)

func main() {
	RunCircuitBreaker()
	time.Sleep(180 * time.Second)
}

func RunCircuitBreaker() {
	log.Println("Machine is starting..")
	successCounter := 0
	failureCounter := 0
	doCounter := 0

	opErrOccured := errors.New("operation error occurred")
	do := func() error {
		doCounter++
		if doCounter%3 == 0 { // operation fails for every three of requests
			log.Println("=> operation failed")
			return opErrOccured
		}
		log.Println("=> operation succeded")
		return nil
	}

	sm := statmach.New(Closed)
	// closed
	scClosed := sm.Configure(Closed)
	scClosed.OnEntryFrom(SuccessThresholdReached, func(...interface{}) {
		log.Println("Entered to Closed state via SuccessThresholdReached..")
		failureCounter = 0
	})
	scClosed.OnEntryFrom(Try, func(params ...interface{}) {
		log.Println("Entered to Closed state via Try..")
		if do() != nil {
			failureCounter++
		}
		if failureCounter >= failureThreshold {
			sm.Fire(FailureThresholdReached)
		} else {
			go FireWithDelay(sm, Try)
		}
	})

	scClosed.Permit(FailureThresholdReached, Open)
	scClosed.PermitReentry(Try)

	// open
	openAnyEntryHandler := func() {
		time.Sleep(2 * time.Second)
		sm.Fire(TimeoutTimerExpired)
	}

	scOpen := sm.Configure(Open)
	scOpen.OnEntryFrom(FailureThresholdReached, func(params ...interface{}) {
		log.Println("Entered to Open state via FailureThresholdReached..")
		openAnyEntryHandler()
	})
	scOpen.OnEntryFrom(OperationFailed, func(...interface{}) {
		log.Println("Entered to Open state via OperationFailed..")
		openAnyEntryHandler()
	})
	scOpen.Permit(TimeoutTimerExpired, HalfOpen)

	// half-open
	halfOpenAnyEntryHandler := func() {
		if do() == nil {
			successCounter++
			if successCounter >= successThreshold {
				sm.Fire(SuccessThresholdReached)
			} else {
				go FireWithDelay(sm, Try)
			}
		} else {
			sm.Fire(OperationFailed)
		}
	}

	scHalfOpen := sm.Configure(HalfOpen)
	scHalfOpen.OnEntryFrom(TimeoutTimerExpired, func(params ...interface{}) {
		log.Println("Entered to Half-Open state via TimeoutTimerExpired..")
		successCounter = 0
		halfOpenAnyEntryHandler()
	})
	scHalfOpen.OnEntryFrom(Try, func(...interface{}) {
		log.Println("Entered to Half-Open state via Try..")
		halfOpenAnyEntryHandler()
	})
	scHalfOpen.Permit(OperationFailed, Open)
	scHalfOpen.Permit(SuccessThresholdReached, Closed)
	scHalfOpen.PermitReentry(Try)

	sm.Fire(Try)
}

func FireWithDelay(sm *statmach.StateMachine, trigger string) {
	time.Sleep(2 * time.Second)
	sm.Fire(trigger)
}
