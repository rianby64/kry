package kry

import (
	"context"
	"errors"
	"fmt"
)

func (fsk *FSM[Action, State, Param]) apply(
	ctx context.Context,
	callbacks callbacks[Action, State, Param],
	action Action,
	from, to State,
	param ...Param,
) error {
	currentHistoryKeeper := fsk.historyKeeper

	currentAction := fsk.currentAction
	currentState := fsk.currentState
	previousState := fsk.previousState

	fsk.currentAction = action
	fsk.currentState = to
	fsk.previousState = currentState
	fsk.runningApply = true

	historyKeeper := newHistoryKeeper[Action, State](
		fsk.historyKeeper.maxLength,
		fsk.stackTrace,
		fsk.cloneHandler,
	)
	fsk.historyKeeper = historyKeeper

	defer func() {
		currentHistoryKeeper.Append(historyKeeper)
		fsk.historyKeeper = currentHistoryKeeper
		fsk.runningApply = false

		if fsk.ignoreCurrent {
			fsk.ignoreCurrent = false

			fsk.currentAction = currentAction
			fsk.currentState = currentState
			fsk.previousState = previousState
		}
	}()

	expectFailed := fsk.decorateWithExpectApply(callbacks)

	if err := fsk.applyTransitionByLengthParams(
		ctx, callbacks, param...,
	); err != nil {
		ignored := fsk.ignoreCurrent
		fsk.ignoreCurrent = true

		if intermediateKeeper, errHistory := fsk.intermediateKeeper(
			historyKeeper,
			action, from, to,
			errors.Unwrap(err), ignored, expectFailed, param...,
		); errHistory != nil {
			err = fmt.Errorf("%w: %w", err, errHistory)
		} else {
			historyKeeper = intermediateKeeper
		}

		return fmt.Errorf("failed to apply (%v) from '%v' to '%v': %w",
			action, from, to, err)
	}

	if intermediateKeeper, errHistory := fsk.intermediateKeeper(
		historyKeeper,
		action, from, to,
		nil, fsk.ignoreCurrent, expectFailed, param...,
	); errHistory != nil {
		return fmt.Errorf("failed to keep forced history: %w", errHistory)
	} else {
		historyKeeper = intermediateKeeper
	}

	return nil
}

func (fsk *FSM[Action, State, Param]) applyByExact(ctx context.Context, action Action, newState State, param ...Param) (bool, error) {
	foundAction := fsk.path[action]
	currentState := fsk.currentState

	foundDstState, ok := foundAction[newState]
	if !ok {
		return false, nil
	}

	callbacks, ok := foundDstState[currentState]
	if !ok {
		return false, nil
	}

	if err := fsk.apply(ctx, callbacks, action, currentState, newState, param...); err != nil {
		return false, err
	}

	return true, nil
}

type matchType int

const (
	matchSrc matchType = iota + 1
	matchDst
)

func (fsk *FSM[Action, State, Param]) applyByMatchSrcDst(ctx context.Context, matchType matchType, action Action, newState State, param ...Param) (bool, error) {
	currentState := fsk.currentState
	var (
		foundActionByMatch map[State][]matchState[Action, State, Param]
		foundStateByMatch  []matchState[Action, State, Param]
		ok                 bool
	)

	switch matchType {
	case matchSrc:
		foundActionByMatch, ok = fsk.pathByMatchSrc[action]
		if !ok {
			return false, nil
		}

		foundStateByMatch, ok = foundActionByMatch[newState]
		if !ok {
			return false, nil
		}

	case matchDst:
		foundActionByMatch, ok = fsk.pathByMatchDst[action]
		if !ok {
			return false, nil
		}

		foundStateByMatch, ok = foundActionByMatch[currentState]
		if !ok {
			return false, nil
		}
	}

	for _, matchState := range foundStateByMatch {
		switch matchType {
		case matchSrc:
			if matchState.MatchSrc(currentState) {
				if err := fsk.apply(ctx, matchState.Callbacks, action, currentState, newState, param...); err != nil {
					return false, err
				}

				return true, nil
			}

		case matchDst:
			if matchState.MatchDst(newState) {
				if err := fsk.apply(ctx, matchState.Callbacks, action, currentState, newState, param...); err != nil {
					return false, err
				}

				return true, nil
			}
		}
	}

	return false, nil
}

func (fsk *FSM[Action, State, Param]) applyByMatch(ctx context.Context, action Action, newState State, param ...Param) (bool, error) {
	foundAction := fsk.pathMatch[action]
	currentState := fsk.currentState

	for _, matchState := range foundAction {
		if matchState.MatchSrc(currentState) && matchState.MatchDst(newState) {
			callbacks := matchState.Callbacks
			if err := fsk.apply(ctx, callbacks, action, currentState, newState, param...); err != nil {
				return false, err
			}

			return true, nil
		}
	}

	return false, nil
}

func (fsk *FSM[Action, State, Param]) applyTransitionByLengthParams(
	ctx context.Context, stateTransition callbacks[Action, State, Param], param ...Param,
) error {
	switch len(param) {
	case 0:
		if stateTransition.EnterNoParams != nil {
			if err := stateTransition.EnterNoParams(ctx, fsk); err != nil {
				return fmt.Errorf("failed to execute enter (no-params) callback: %w", err)
			}

			return nil
		}

	case 1:
		if stateTransition.Enter != nil {
			if err := stateTransition.Enter(ctx, fsk, param[0]); err != nil {
				return fmt.Errorf("failed to execute enter (single-param) callback: %w", err)
			}

			return nil
		}
	}

	if stateTransition.EnterVariadic != nil {
		if err := stateTransition.EnterVariadic(ctx, fsk, param...); err != nil {
			return fmt.Errorf("failed to execute enter (variadic) callback: %w", err)
		}

		return nil
	}

	return nil
}

func (fsk *FSM[Action, State, Param]) Event(
	ctx context.Context, action Action, param ...Param,
) error {
	if !fsk.canTriggerEvents {
		return fmt.Errorf("event %v: %w", action, ErrNotAllowed)
	}

	foundEvent, ok := fsk.events[action]
	if !ok {
		return fmt.Errorf("event %w: %v", ErrUnknown, action)
	}

	newState := foundEvent.Dst

	if err := fsk.Apply(ctx, action, newState, param...); err != nil {
		return fmt.Errorf("failed to apply event %v: %w", action, err)
	}

	return nil
}

func (fsk *FSM[Action, State, Param]) Apply(
	ctx context.Context, action Action, newState State, param ...Param,
) error {
	currentState := fsk.currentState

	defer func() {
		if errPanic := recover(); errPanic != nil {
			defer func() {
				fsk.currentState = currentState // rollback state
			}()

			err := fmt.Errorf("%v", errPanic)

			if errHistory := fsk.historyKeeper.Push(
				action, currentState, newState,
				err, defaultSkipStackTrace, fsk.ignoreCurrent, false,
				param...,
			); errHistory != nil {
				err = fmt.Errorf("%w: failed to push history item: %w", err, errHistory)
			}

			if fsk.panicHandler != nil {
				fsk.panicHandler(ctx, errPanic)

				return
			}

			panic(err)
		}
	}()

	ctxWithLoop, err := fsk.checkLoop(ctx, currentState, newState)
	if err != nil {
		return fmt.Errorf("failed to apply (%v): %w", action, err)
	}

	if _, ok := fsk.path[action]; !ok {
		err = ErrUnknown
		if errHistory := fsk.historyKeeper.Push(
			action, currentState, newState,
			err, defaultSkipStackTrace, fsk.ignoreCurrent, false,
			param...,
		); errHistory != nil {
			err = fmt.Errorf("%w: failed to push history item: %w", err, errHistory)
		}

		return fmt.Errorf("action %w: %v", err, action)
	}

	if applied, err := fsk.applyByExact(ctxWithLoop, action, newState, param...); err != nil {
		return err
	} else if applied {
		return nil
	}

	if applied, err := fsk.applyByMatchSrcDst(ctxWithLoop, matchSrc, action, newState, param...); err != nil {
		return err
	} else if applied {
		return nil
	}

	if applied, err := fsk.applyByMatchSrcDst(ctxWithLoop, matchDst, action, newState, param...); err != nil {
		return err
	} else if applied {
		return nil
	}

	if applied, err := fsk.applyByMatch(ctxWithLoop, action, newState, param...); err != nil {
		return err
	} else if applied {
		return nil
	}

	err = ErrNotFound
	if errHistory := fsk.historyKeeper.Push(
		action, currentState, newState,
		err, defaultSkipStackTrace, fsk.ignoreCurrent, false,
		param...,
	); errHistory != nil {
		err = fmt.Errorf("%w: failed to push history item: %w", err, errHistory)
	}

	return fmt.Errorf("transition (%v) from state %v: %w", action, currentState, err)
}
