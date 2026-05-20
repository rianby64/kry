package kry

import "fmt"

var (
	idMachine uint64
)

func callbacksFrom[State comparable, Param any](name string, h TransitionHandler[State, Param]) callbacks[State, Param] {
	cbs := callbacks[State, Param]{Name: name}
	switch h.arity {
	case arityWith:
		cbs.Enter = h.with
	case arityVariadic:
		cbs.EnterVariadic = h.variadic
	default:
		cbs.EnterNoParams = h.noParams
	}
	return cbs
}

func constructFromTransitions[State comparable, Param any](
	initialState State,
	transitions []Transition[State, Param],
) (
	map[State]map[State]callbacks[State, Param],
	map[State][]matchState[State, Param],
	map[State][]matchState[State, Param],
	[]matchState[State, Param],
	map[State]struct{},
	error,
) {
	path := make(map[State]map[State]callbacks[State, Param])
	pathByMatchSrc := make(map[State][]matchState[State, Param])
	pathByMatchDst := make(map[State][]matchState[State, Param])
	pathMatch := make([]matchState[State, Param], 0)
	states := map[State]struct{}{initialState: {}}

	var zeroState State

	for index, transition := range transitions {
		name := transition.Name

		if len(transition.Src) == 0 && transition.SrcFn != nil && transition.DstFn != nil && transition.Dst == zeroState {
			pathMatch = append(pathMatch, matchState[State, Param]{
				MatchSrc: transition.SrcFn,
				MatchDst: transition.DstFn,
				Callbacks: callbacksFrom(name, transition.Enter),
			})

			continue
		}

		if len(transition.Src) == 0 && transition.SrcFn == nil {
			return nil, nil, nil, nil, nil,
				fmt.Errorf("for transition %q(index=%d) neither src states nor matching function found: %w", name, index, ErrNotFound)
		}

		dst := transition.Dst

		if transition.DstFn != nil {
			for _, src := range transition.Src {
				if _, ok := pathByMatchDst[src]; !ok {
					pathByMatchDst[src] = make([]matchState[State, Param], 0)
				}

				states[src] = struct{}{}
				pathByMatchDst[src] = append(pathByMatchDst[src], matchState[State, Param]{
					MatchDst:  transition.DstFn,
					Callbacks: callbacksFrom(name, transition.Enter),
				})
			}
		}

		if dst == zeroState && transition.DstFn != nil {
			continue
		}

		if _, ok := path[dst]; !ok {
			path[dst] = make(map[State]callbacks[State, Param])
		}

		if transition.SrcFn != nil {
			if _, ok := pathByMatchSrc[dst]; !ok {
				pathByMatchSrc[dst] = make([]matchState[State, Param], 0)
			}

			pathByMatchSrc[dst] = append(pathByMatchSrc[dst], matchState[State, Param]{
				MatchSrc: transition.SrcFn,
				Callbacks: callbacksFrom(name, transition.Enter),
			})
		}

		for _, src := range transition.Src {
			if _, ok := path[dst][src]; ok {
				return nil, nil, nil, nil, nil,
					fmt.Errorf(
						"transition %q from state %v to state %v: %w",
						name, src, dst, ErrRepeated,
					)
			}

			states[src] = struct{}{}
			path[dst][src] = callbacksFrom(name, transition.Enter)
		}

		states[dst] = struct{}{}
	}

	return path, pathByMatchSrc, pathByMatchDst, pathMatch, states, nil
}
