# statmach
yet another hierarchical state machine in Go. it is
mostly influenced by [dotnet-state-machine/stateless](https://github.com/dotnet-state-machine/stateless) :heart:

## Features
- Hierarchical states
- Conditional transitions
- Parametric triggers
- Entry/exit events for states

## Usage

The basic example of Circuit Breaker:

<img src="img/circuit-breaker-diagram.png" width="480" title="Circuit Breaker Diagram" />

```
...
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
```