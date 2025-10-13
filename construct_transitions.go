package kry

import "fmt"

var (
	idMachine uint64
)

func constructFromTransitions[Action, State comparable, Param any](
	initialState State,
	transitions []Transition[Action, State, Param],
) (
	map[Action]map[State]map[State]callbacks[Action, State, Param],
	map[Action]map[State][]matchState[Action, State, Param],
	map[State]struct{},
	map[Action]Transition[Action, State, Param],
	bool,
	error,
) {
	path := make(map[Action]map[State]map[State]callbacks[Action, State, Param])
	pathByMatch := make(map[Action]map[State][]matchState[Action, State, Param])
	states := map[State]struct{}{initialState: {}}
	canTriggerEvents := true
	events := make(map[Action]Transition[Action, State, Param])

	for _, transition := range transitions {
		action := transition.Name
		if _, ok := path[action]; !ok {
			path[action] = make(map[State]map[State]callbacks[Action, State, Param])
		}

		if _, ok := events[action]; ok {
			canTriggerEvents = false
		}

		if len(transition.Src) == 0 && transition.Match == nil {
			return nil, nil, nil, nil, false,
				fmt.Errorf("for action %v neither src states nor matching function found: %w", action, ErrNotFound)
		}

		dst := transition.Dst
		if _, ok := path[action][dst]; !ok {
			path[action][dst] = make(map[State]callbacks[Action, State, Param])
		}

		if transition.Match != nil {
			if _, ok := pathByMatch[action]; !ok {
				pathByMatch[action] = make(map[State][]matchState[Action, State, Param])
			}

			if _, ok := pathByMatch[action][dst]; !ok {
				pathByMatch[action][dst] = make([]matchState[Action, State, Param], 0)
			}

			pathByMatch[action][dst] = append(pathByMatch[action][dst], matchState[Action, State, Param]{
				Match: transition.Match,
				Callbacks: callbacks[Action, State, Param]{
					EnterVariadic: transition.EnterVariadic,
					Enter:         transition.Enter,
					EnterNoParams: transition.EnterNoParams,
				},
			})
		}

		for _, src := range transition.Src {
			if _, ok := path[action][dst][src]; ok {
				return nil, nil, nil, nil, false,
					fmt.Errorf(
						"action %v from state %v to state %v: %w",
						action, src, dst, ErrRepeated,
					)
			}

			path[action][dst][src] = callbacks[Action, State, Param]{
				EnterVariadic: transition.EnterVariadic,
				Enter:         transition.Enter,
				EnterNoParams: transition.EnterNoParams,
			}
		}

		for _, state := range transition.Src {
			states[state] = struct{}{}
		}

		events[action] = transition
		states[transition.Dst] = struct{}{}
	}

	return path,
		pathByMatch,
		states,
		events,
		canTriggerEvents,
		nil
}
