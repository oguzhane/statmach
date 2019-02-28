# statmach
yet another hierarchical state machine in Go. it is
mostly influenced by [dotnet-state-machine/stateless](https://github.com/dotnet-state-machine/stateless) :heart:

## Features
- Hierarchical states
- Conditional transitions
- Parametric triggers
- Entry/exit events for states

## Example Usage
```
package main

import (
	"fmt"

	"github.com/oguzhane/statmach"
)

// STATES
const (
	EmptyForm    = "EmptyForm"
	Submitting   = "Submitting"
	ErrorPage    = "ErrorPage"
	WelcomePage  = "WelcomePage"
	LoginPage    = "LoginPage"
	MySuperState = "MySuperState"
)

// TRIGGERS
const (
	SUBMIT   = "SUBMIT"
	REGISTER = "REGISTER"
	REJECT   = "REJECT"
	RESOLVE  = "RESOLVE"
	LOGOUT   = "LOGOUT"
)

func main() {

	sm := statmach.New(EmptyForm)

	sc := sm.Configure(EmptyForm)
	err := sc.Permit(SUBMIT, Submitting)
	Fatal(err)

	sc = sm.Configure(Submitting)
	err = sc.Permit(REJECT, ErrorPage)
	Fatal(err)
	err = sc.Permit(RESOLVE, WelcomePage)
	Fatal(err)

	err = sm.Configure(ErrorPage).SubstateOf(MySuperState)
	Fatal(err)

	sc = sm.Configure(WelcomePage)
	err = sc.SubstateOf(MySuperState)
	Fatal(err)
	err = sc.Permit(LOGOUT, LoginPage)
	Fatal(err)

	err = sm.Configure(LoginPage).SubstateOf(MySuperState)
	Fatal(err)

	sc = sm.Configure(MySuperState)
	err = sc.Permit(REGISTER, EmptyForm)
	Fatal(err)

	sm.Fire(SUBMIT)
	sm.Fire(REJECT)

	fmt.Println(sm.GetCurrentState().GetStateName())
	sm.Fire(REGISTER)
	fmt.Println(sm.GetCurrentState().GetStateName())
	_, err = sm.Fire(LOGOUT) // transition from EmptyForm through LOGOUT is not allowed
	fmt.Println(err)
}

func Fatal(err error) {
	if err != nil {
		panic(err)
	}
}
```