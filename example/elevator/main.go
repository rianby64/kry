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

// The state set – every possible "elevator position".
type ElevatorState string

const (
	Stop  ElevatorState = "stop"
	Up    ElevatorState = "up"
	Down  ElevatorState = "down"
	Start ElevatorState = "start" // entry point; only used for the demo
)

// -----------------------------------------------------------------
// 2️⃣  Transitions  (the "real work" that will be validated later)
// -----------------------------------------------------------------
func NewElevator() (*kry.FSM[ElevatorState, ElevatorParam], error) {
	fsk, err := kry.Build[ElevatorState, ElevatorParam](Start).
		GoingTo(Up,
			kry.From[ElevatorState, ElevatorParam](Stop).On("up", kry.OnEnter(func(ctx context.Context, i kry.InstanceFSM[ElevatorState, ElevatorParam], p ElevatorParam) error {
				fmt.Printf("🛑 → %s  (expected → floor %d)\n", i.Current(), p.Floor)
				return nil
			})),
		).
		GoingTo(Down,
			kry.From[ElevatorState, ElevatorParam](Stop).On("down", kry.OnEnter(func(ctx context.Context, i kry.InstanceFSM[ElevatorState, ElevatorParam], p ElevatorParam) error {
				fmt.Printf("↓  → floor %d\n", p.Floor)
				return nil
			})),
		).
		GoingTo(Stop,
			kry.From[ElevatorState, ElevatorParam](Up, Down, Start).On("stop", kry.NoOp[ElevatorState, ElevatorParam]()),
		).
		WithFullHistory().
		Create()

	if err != nil {
		panic(err)
	}

	return fsk, nil
}

// -----------------------------------------------------------------
// 4️⃣  Running the FSM
// -----------------------------------------------------------------
func main() {
	ctx := context.Background()

	fsk, err := NewElevator()
	if err != nil {
		panic(err)
	}

	// Transition from Start → Stop first
	if err := fsk.Apply(ctx, Stop, ElevatorParam{Floor: 0}); err != nil {
		panic(err)
	}

	// Now go Up from Stop
	if err := fsk.Apply(ctx, Up, ElevatorParam{Floor: 5}); err != nil {
		panic(err)
	}

	// Back to Stop
	if err := fsk.Apply(ctx, Stop, ElevatorParam{Floor: 0}); err != nil {
		panic(err)
	}

	// Go Down from Stop
	if err := fsk.Apply(ctx, Down, ElevatorParam{Floor: 4}); err != nil {
		panic(err)
	}

	// Back to Stop
	if err := fsk.Apply(ctx, Stop, ElevatorParam{Floor: 0}); err != nil {
		panic(err)
	}

	fmt.Println("Current engine:", fsk.Current())
}
