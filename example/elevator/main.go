package main

import (
	"context"
	"fmt"

	"github.com/rianby64/kry"
)

// 1️⃣  Parameters that flow through the FSM.
type ElevatorParam struct {
	Floor int // the current floor the car is on
}

// The state set – every possible “elevator position”.
type ElevatorState string

const (
	Stop  ElevatorState = "stop"
	Up    ElevatorState = "up"
	Down  ElevatorState = "down"
	Start ElevatorState = "start" // entry point; only used for the demo
)

// -----------------------------------------------------------------
// 2️⃣  Transitions  (the “real work” that will be validated later)
// -----------------------------------------------------------------
func NewElevator() (*kry.FSM[string, ElevatorState, ElevatorParam], error) {
	// The builder – it will create the FSM and also give us a slot
	// where decorators can be placed (`decoratorApply`).
	fsk, err := kry.New(Start, []kry.Transition[string, ElevatorState, ElevatorParam]{
		// -------------------------------------------------------------
		//  Upward movement – comes from Stop (or Down) and goes to Up.
		//  A callback is expected *only* when a “up” event fires.
		// -------------------------------------------------------------
		{
			Name: "up",
			Dst:  Up,
			Src:  []ElevatorState{Stop},
			Enter: func(ctx context.Context, i kry.InstanceFSM[string, ElevatorState, ElevatorParam], p ElevatorParam) error {
				fmt.Printf("🛑 → %s  (expected → floor %d)\n", i.Current(), p.Floor)
				// The library already called the decorator’s expectation,
				// we just verify it inside the callback.
				return nil
			},
			// No extra callback when we leave “stop”, because we always
			// go up.  The library will see that `expectToCallEnter` got a
			// non‑nil slice and flag it as an expectation.
		},

		// -------------------------------------------------------------
		//  Downward movement – from Stop → Down, expects a “down” callback
		//  (no parameters needed).
		// -------------------------------------------------------------
		{
			Name: "down",
			Dst:  Down,
			Src:  []ElevatorState{Stop},
			Enter: func(ctx context.Context, i kry.InstanceFSM[string, ElevatorState, ElevatorParam], p ElevatorParam) error {
				fmt.Printf("↓  → floor %d\n", p.Floor)
				return nil
			},
			// expectation = one handler that expects no parameters.
		},

		// -------------------------------------------------------------
		//  Stop – we can stop from any state, but we *expect* an
		//  “stop” entry‑callback (no parameters).  The validator checks it.
		// -------------------------------------------------------------
		{
			Name: "stop",
			Dst:  Stop,
			Src:  []ElevatorState{Stop, Up, Down},
			// No callback right now – it will be added later by
			// `WithExpectations()` –‑‑ this is the part where the library
			// populates `decoratorApply.expectToCallEnter`.
		},
	}, kry.WithFullHistory[ElevatorParam]())

	if err != nil {
		panic(err)
	}

	return fsk, nil
}

// floorForState resolves a state to a floor (used for params)
func floorForState(s ElevatorState) int {
	switch s {
	case Up:
		return 3
	case Down:
		return 1
	default:
		return 0
	}
}

// -----------------------------------------------------------------
// 4️⃣  Running the FSM – events, apply, callbacks validated
// -----------------------------------------------------------------
func main() {
	ctx := context.Background()

	// 4️⃣.1 Create the FSM (the validator runs during construction).
	fsk, err := NewElevator()
	if err != nil {
		panic(err)
	}

	// 4️⃣.2 Apply the internal decorator – this tells the library to
	//      check its expectations before any transitions are allowed.
	// if err := fsk.WithDecorator(decoratorApply{}), err != nil {
	//	panic(err)
	// }

	// -----------------------------------------------------------------
	// 4️⃣.3 Register the *external* event streams.
	// -----------------------------------------------------------------
	if err := fsk.With().Event(ctx, "up", ElevatorParam{Floor: 5}); err != nil {
		panic(err)
	}
	if err := fsk.With().Event(ctx, "down", ElevatorParam{Floor: 4}); err != nil {
		panic(err)
	}

	// -----------------------------------------------------------------
	// 4️⃣.4 Push some “internal” transitions – the validator checks
	//      that an “enter” callback was expected for the transition to
	//      `stop`.  Because we added a dummy entry callback,
	//      the validator will succeed.
	// -----------------------------------------------------------------
	if err := fsk.Apply(ctx, "stop", Stop, ElevatorParam{Floor: 0}); err != nil {
		panic(err)
	}

	// -----------------------------------------------------------------
	// 4️⃣.5 Final state – we should be at Stop with no error.
	// -----------------------------------------------------------------
	fmt.Println("Current engine:", fsk.Current())
}
