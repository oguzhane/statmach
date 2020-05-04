package statmach

import (
	"errors"
	"fmt"
)

type transitionRepresentation struct {
	guardFunc func(params ...interface{}) bool
	trigger   string
	destState *StateConfigure
}

func newTransitionRepresentation(destState *StateConfigure, trigger string, guardFunc func(params ...interface{}) bool) *transitionRepresentation {
	return &transitionRepresentation{
		destState: destState,
		guardFunc: guardFunc,
		trigger:   trigger,
	}
}

// StateConfigure is used to represent a state and its transitions
type StateConfigure struct {
	name          string
	sm            *StateMachine
	transitionMap map[string]*transitionRepresentation
	parentState   *StateConfigure
	substates     map[string]*StateConfigure
	onExitFunc    func(trigger string, destState string)
	onEntryMap    map[string]func(params ...interface{})
}

func newStateConfigure(name string, sm *StateMachine) *StateConfigure {
	return &StateConfigure{
		name:          name,
		sm:            sm,
		transitionMap: make(map[string]*transitionRepresentation),
		onEntryMap:    make(map[string]func(params ...interface{})),
		substates:     make(map[string]*StateConfigure),
	}
}

// Name returns the state name
func (c *StateConfigure) Name() string {
	return c.name
}

func (c *StateConfigure) internalPermit(trigger string, destState string, guardFunc func(params ...interface{}) bool) error {
	if c.name == destState {
		return errors.New("Destination state cannot be the same as name of the state by Permit method.Try to use PermitReentry")
	}
	transRepresent, ok := c.transitionMap[trigger]
	if ok {
		return fmt.Errorf("a transition between from %v to %v via '%v' already exists", c.name, transRepresent.destState.name, trigger)
	}
	// append transition to map
	transRepresent = newTransitionRepresentation(c.sm.Configure(destState), trigger, guardFunc)
	c.transitionMap[trigger] = transRepresent
	return nil
}

// Permit adds a transition to destState via trigger
func (c *StateConfigure) Permit(trigger string, destState string) error {
	return c.internalPermit(trigger, destState, nil)
}

// PermitIf adds a conditional transition to destState via trigger.
// transition occurs if it is allowed by guardFunc
func (c *StateConfigure) PermitIf(trigger string, destState string, guardFunc func(params ...interface{}) bool) error {
	if guardFunc == nil {
		return errors.New("guardFunc cannot be nil")
	}
	return c.internalPermit(trigger, destState, guardFunc)
}

func (c *StateConfigure) internalPermitReentry(trigger string, guardFunc func(params ...interface{}) bool) error {
	transRepresent, ok := c.transitionMap[trigger]
	if ok {
		return fmt.Errorf("a transition between from %v to %v via '%v' already exists", c.name, transRepresent.destState.name, trigger)
	}
	c.transitionMap[trigger] = newTransitionRepresentation(c, trigger, guardFunc)
	return nil
}

// PermitReentry adds a transition to itself via trigger
func (c *StateConfigure) PermitReentry(trigger string) error {
	return c.internalPermitReentry(trigger, nil)
}

// PermitReentryIf adds a conditional transition to itself via trigger.
// transition occurs if it is allowed by guardFunc
func (c *StateConfigure) PermitReentryIf(trigger string, guardFunc func(params ...interface{}) bool) error {
	if guardFunc == nil {
		return errors.New("guardFunc cannot be nil")
	}
	return c.internalPermitReentry(trigger, guardFunc)
}

// OnEntryFrom registers entry handler for the specified trigger when the current state changes to the state
func (c *StateConfigure) OnEntryFrom(trigger string, handlerFn func(params ...interface{})) error {
	if handlerFn == nil {
		return errors.New("onEntryFrom handler cannot be nil")
	}
	_, ok := c.onEntryMap[trigger]
	if ok {
		return errors.New("a function to handle entry transition for the trigger is already registered")
	}
	c.onEntryMap[trigger] = handlerFn
	return nil
}

// OnExit registers exit handler for the specified trigger when the machine leaves the current state
func (c *StateConfigure) OnExit(fn func(trigger string, destState string)) error {
	if c.onExitFunc != nil {
		return errors.New("onExit can be handled by only just one function")
	}
	c.onExitFunc = fn
	return nil
}

// SubstateOf registers the state as substate of parent state
func (c *StateConfigure) SubstateOf(parentStateName string) error {
	if c.parentState != nil {
		return errors.New("a state could have just only one parent")
	}
	if c.name == parentStateName {
		return errors.New("a state cannot be substate of itself")
	}

	if _, ok := c.substates[parentStateName]; ok {
		return errors.New("states cannot be substates of each other")
	}

	c.parentState = c.sm.Configure(parentStateName)
	c.parentState.substates[c.name] = c

	return nil
}

// StateMachine represents a state machine
type StateMachine struct {
	stateMap     map[string]*StateConfigure
	currentState *StateConfigure
}

// New creates new state machine
func New(initialState string) *StateMachine {
	sm := &StateMachine{
		stateMap: make(map[string]*StateConfigure),
	}
	sm.currentState = sm.Configure(initialState)
	return sm
}

// Configure adds a state into the machine. If the state already exists, returns the existing one
func (sm *StateMachine) Configure(stateName string) *StateConfigure {
	sc, ok := sm.stateMap[stateName]
	if ok {
		return sc
	}
	sc = newStateConfigure(stateName, sm)
	sm.stateMap[stateName] = sc
	return sc
}

// CurrentState returns the current state of the machine
func (sm *StateMachine) CurrentState() *StateConfigure {
	return sm.currentState
}

func (sm *StateMachine) lookUpTransition(trigger string, sourceState *StateConfigure) (transition *transitionRepresentation, srcState *StateConfigure, err error) {
	currState := sourceState
	transRepresent, ok := currState.transitionMap[trigger]
	for {
		if ok {
			return transRepresent, currState, nil
		}
		if currState.parentState == nil {
			break
		}
		currState = currState.parentState
		transRepresent, ok = currState.transitionMap[trigger]
	}
	return nil, nil, errors.New("a valid transition not found")
}

// Fire triggers off a transition from the current state via trigger
// params will be passed to exit and entry handlers
func (sm *StateMachine) Fire(trigger string, params ...interface{}) (bool, error) {
	transRepresent, _, errValidTransition := sm.lookUpTransition(trigger, sm.currentState)
	if errValidTransition != nil {
		return false, errValidTransition
	}
	allowTransition := true
	if transRepresent.guardFunc != nil {
		allowTransition = transRepresent.guardFunc(params...)
	}
	if allowTransition {
		destState := transRepresent.destState
		if sm.currentState.onExitFunc != nil {
			sm.currentState.onExitFunc(trigger, transRepresent.destState.name)
		}
		sm.currentState = destState // update current state
		if entryHandler, entryOk := destState.onEntryMap[trigger]; entryOk {
			entryHandler(params...)
		}
	}
	return allowTransition, nil
}
