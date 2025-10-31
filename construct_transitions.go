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
	map[Action]map[State][]matchState[Action, State, Param],
	map[Action][]matchState[Action, State, Param],
	map[State]struct{},
	map[Action]Transition[Action, State, Param],
	bool,
	error,
) {
	path := make(map[Action]map[State]map[State]callbacks[Action, State, Param])
	pathByMatchSrc := make(map[Action]map[State][]matchState[Action, State, Param])
	pathByMatchDst := make(map[Action]map[State][]matchState[Action, State, Param])
	pathMatch := make(map[Action][]matchState[Action, State, Param])
	states := map[State]struct{}{initialState: {}}
	canTriggerEvents := true
	events := make(map[Action]Transition[Action, State, Param])

	var zeroState State

	for index, transition := range transitions {
		action := transition.Name
		if _, ok := path[action]; !ok {
			path[action] = make(map[State]map[State]callbacks[Action, State, Param])
		}

		if _, ok := events[action]; ok {
			canTriggerEvents = false
		}

		if len(transition.Src) == 0 && transition.SrcFn != nil && transition.DstFn != nil && transition.Dst == zeroState {
			if _, ok := pathMatch[action]; !ok {
				pathMatch[action] = make([]matchState[Action, State, Param], 0)
			}

			pathMatch[action] = append(pathMatch[action], matchState[Action, State, Param]{
				MatchSrc: transition.SrcFn,
				MatchDst: transition.DstFn,
				Callbacks: callbacks[Action, State, Param]{
					EnterVariadic: transition.EnterVariadic,
					Enter:         transition.Enter,
					EnterNoParams: transition.EnterNoParams,
				},
			})

			continue
		}

		if len(transition.Src) == 0 && transition.SrcFn == nil {
			return nil, nil, nil, nil, nil, nil, false,
				fmt.Errorf("for action %v(index=%d) neither src states nor matching function found: %w", action, index, ErrNotFound)
		}

		dst := transition.Dst
		if dst == zeroState && transition.DstFn == nil {
			return nil, nil, nil, nil, nil, nil, false,
				fmt.Errorf("for action %v(index=%d) destination state is zero value: %w", action, index, ErrNotAllowed)
		}

		if transition.DstFn != nil {
			if _, ok := pathByMatchDst[action]; !ok {
				pathByMatchDst[action] = make(map[State][]matchState[Action, State, Param])
			}

			for _, src := range transition.Src {
				if _, ok := pathByMatchDst[action][src]; !ok {
					pathByMatchDst[action][src] = make([]matchState[Action, State, Param], 0)
				}

				states[src] = struct{}{}
				pathByMatchDst[action][src] = append(pathByMatchDst[action][src], matchState[Action, State, Param]{
					MatchDst: transition.DstFn,
					Callbacks: callbacks[Action, State, Param]{
						EnterVariadic: transition.EnterVariadic,
						Enter:         transition.Enter,
						EnterNoParams: transition.EnterNoParams,
					},
				})
			}
		}

		if dst == zeroState {
			continue
		}

		if _, ok := path[action][dst]; !ok {
			path[action][dst] = make(map[State]callbacks[Action, State, Param])
		}

		if transition.SrcFn != nil {
			if _, ok := pathByMatchSrc[action]; !ok {
				pathByMatchSrc[action] = make(map[State][]matchState[Action, State, Param])
			}

			if _, ok := pathByMatchSrc[action][dst]; !ok {
				pathByMatchSrc[action][dst] = make([]matchState[Action, State, Param], 0)
			}

			pathByMatchSrc[action][dst] = append(pathByMatchSrc[action][dst], matchState[Action, State, Param]{
				MatchSrc: transition.SrcFn,
				Callbacks: callbacks[Action, State, Param]{
					EnterVariadic: transition.EnterVariadic,
					Enter:         transition.Enter,
					EnterNoParams: transition.EnterNoParams,
				},
			})
		}

		for _, src := range transition.Src {
			if _, ok := path[action][dst][src]; ok {
				return nil, nil, nil, nil, nil, nil, false,
					fmt.Errorf(
						"action %v from state %v to state %v: %w",
						action, src, dst, ErrRepeated,
					)
			}

			states[src] = struct{}{}
			path[action][dst][src] = callbacks[Action, State, Param]{
				EnterVariadic: transition.EnterVariadic,
				Enter:         transition.Enter,
				EnterNoParams: transition.EnterNoParams,
			}
		}

		events[action] = transition
		states[dst] = struct{}{}
	}

	return path,
		pathByMatchSrc,
		pathByMatchDst,
		pathMatch,
		states,
		events,
		canTriggerEvents,
		nil
}
