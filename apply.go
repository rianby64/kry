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
	currentState, newState State,
	param ...Param,
) error {
	oldForcedHistoryKeeper := fsk.forcedHistoryKeeper
	oldHistoryKeeper := fsk.historyKeeper
	oldAction := fsk.currentAction

	fsk.currentAction = action
	fsk.currentState = newState

	historyKeeper := newHistoryKeeper[Action, State, Param](
		fsk.historyKeeper.maxLength,
		fsk.stackTrace,
	)
	forcedHistoryKeeper := newHistoryKeeper[Action, State, Param](
		fsk.historyKeeper.maxLength,
		fsk.stackTrace,
	)
	fsk.historyKeeper = historyKeeper
	fsk.forcedHistoryKeeper = forcedHistoryKeeper

	defer func() {
		oldHistoryKeeper.Append(oldForcedHistoryKeeper)
		oldHistoryKeeper.Append(historyKeeper)
		fsk.historyKeeper = oldHistoryKeeper
		oldForcedHistoryKeeper.Clear()
		fsk.forcedHistoryKeeper = newHistoryKeeper[Action, State, Param](
			fsk.historyKeeper.maxLength,
			fsk.stackTrace,
		)
	}()

	if err := fsk.switchEventByLengthParams(ctx, callbacks, param...); err != nil {
		fsk.currentAction = oldAction
		fsk.currentState = currentState

		if intermediateHistory, errHistory := fsk.keepForcedHistory(
			forcedHistoryKeeper,
			action,
			currentState,
			newState,
			errors.Unwrap(err),
			param...,
		); errHistory != nil {
			err = fmt.Errorf("%w: %w", err, errHistory)
		} else {
			historyKeeper = intermediateHistory
		}

		return fmt.Errorf("failed to apply (%v) from '%v' to '%v': %w",
			action, currentState, newState, err)
	}

	if intermediateHistory, errHistory := fsk.keepForcedHistory(
		forcedHistoryKeeper,
		action,
		currentState,
		newState,
		nil,
		param...,
	); errHistory != nil {
		return fmt.Errorf("failed to keep forced history: %w", errHistory)
	} else {
		historyKeeper = intermediateHistory
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

func (fsk *FSM[Action, State, Param]) applyByMatch(ctx context.Context, action Action, newState State, param ...Param) (bool, error) {
	currentState := fsk.currentState
	foundActionByMatch, ok := fsk.pathByMatch[action]
	if !ok {
		return false, nil
	}

	foundDstByMatch, ok := foundActionByMatch[newState]
	if !ok {
		return false, nil
	}

	for _, matchState := range foundDstByMatch {
		if matchState.Match(currentState) {
			if err := fsk.apply(ctx, matchState.Callbacks, action, currentState, newState, param...); err != nil {
				return false, err
			}

			return true, nil
		}
	}

	return false, nil
}

func (fsk *FSM[Action, State, Param]) switchEventByLengthParams(ctx context.Context, stateTransition callbacks[Action, State, Param], param ...Param) error {
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

func (fsk *FSM[Action, State, Param]) Event(ctx context.Context, action Action, param ...Param) error {
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

func (fsk *FSM[Action, State, Param]) Apply(ctx context.Context, action Action, newState State, param ...Param) error {
	currentState := fsk.currentState

	ctxWithLoop, err := fsk.checkLoop(ctx, currentState, newState)
	if err != nil {
		return fmt.Errorf("failed to apply (%v): %w", action, err)
	}

	if _, ok := fsk.path[action]; !ok {
		return fmt.Errorf("action %w: %v", ErrUnknown, action)
	}

	if applied, err := fsk.applyByExact(ctxWithLoop, action, newState, param...); err != nil {
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
	errHistory := fsk.historyKeeper.Push(action, currentState, newState, ErrNotFound, param...)
	if errHistory != nil {
		err = fmt.Errorf("%w: failed to push history item: %w", err, errHistory)
	}

	return fmt.Errorf("transition (%v) from state %v: %w", action, currentState, err)
}
