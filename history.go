package kry

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

const (
	fullHistorySize = -1
)

type HistoryItem[Action, State comparable, Param any] struct {
	Forced bool
	Action Action
	From   State
	To     State
	Params []Param
	Error  error
}

type historyItem[Action, State comparable, Param any] struct {
	*HistoryItem[Action, State, Param]
	Next *historyItem[Action, State, Param]
}

type historyKeeper[Action, State comparable, Param any] struct {
	maxLength int
	head      *historyItem[Action, State, Param]
	tail      *historyItem[Action, State, Param]
	length    int
}

func newHistoryKeeper[Action, State comparable, Param any](size int) *historyKeeper[Action, State, Param] {
	return &historyKeeper[Action, State, Param]{
		maxLength: size,
		head:      nil,
		tail:      nil,
		length:    0,
	}
}

func cloneParams[Param any](params ...Param) ([]Param, error) {
	if params == nil {
		return nil, nil
	}

	data, err := cbor.Marshal(params)
	if err != nil {
		return nil, err
	}

	var cloned []Param
	if err := cbor.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}

	return cloned, nil
}

func (hk *historyKeeper[Action, State, Param]) Push(action Action, from State, to State, err error, params ...Param) error {
	return hk.push(action, from, to, err, false, params...)
}

func (hk *historyKeeper[Action, State, Param]) PushForced(action Action, from State, to State, err error, params ...Param) error {
	return hk.push(action, from, to, err, true, params...)
}

func (hk *historyKeeper[Action, State, Param]) push(action Action, from State, to State, err error, forced bool, params ...Param) error {
	if hk.maxLength == 0 {
		return nil
	}

	cloneParams, errClone := cloneParams(params...)
	if errClone != nil {
		return errClone
	}

	item := &historyItem[Action, State, Param]{
		HistoryItem: &HistoryItem[Action, State, Param]{
			Action: action,
			From:   from,
			To:     to,
			Params: cloneParams,
			Error:  err,
			Forced: forced,
		},
	}

	if hk.maxLength > 0 && hk.length >= hk.maxLength {
		hk.tail.Next = item
		hk.tail = item
		hk.head = hk.head.Next

		return nil
	}

	hk.length++
	if hk.head == nil {
		hk.head = item
		hk.tail = item
	} else {
		hk.tail.Next = item
		hk.tail = item
	}

	return nil
}

func (hk *historyKeeper[Action, State, Param]) Items() []HistoryItem[Action, State, Param] {
	items := make([]HistoryItem[Action, State, Param], hk.length)

	i := 0
	current := hk.head
	for current != nil {
		items[i] = *current.HistoryItem
		current = current.Next
		i++
	}

	return items
}

func (hk *historyKeeper[Action, State, Param]) Append(other *historyKeeper[Action, State, Param]) {
	hk.tail.Next = other.head
	hk.tail = other.tail
	hk.length += other.length

	if hk.maxLength > 0 && hk.length > hk.maxLength { // TODO: test this
		excess := hk.length - hk.maxLength
		current := hk.head

		for range excess {
			current = current.Next
		}

		hk.head = current
		hk.length = hk.maxLength
	}
}

func (hk *historyKeeper[Action, State, Param]) Clear() {
	hk.head = nil
	hk.tail = nil
	hk.length = 0
}

// the following methods are added to FSM because they relate to history management

func (fsk *FSM[Action, State, Param]) keepForcedHistory(
	action Action,
	currentState, newState State,
	err error,
	param ...Param,
) error {
	if fsk.forcedHistoryKeeper.length > 0 {
		fsk.historyKeeper.Append(fsk.forcedHistoryKeeper)
		fsk.forcedHistoryKeeper.Clear()
	}

	if errHistory := fsk.historyKeeper.
		Push(action, currentState, newState, err, param...); errHistory != nil {
		return fmt.Errorf("failed to push history item: %w", errHistory)
	}

	return nil
}

func (fsk *FSM[Action, State, Param]) History() []HistoryItem[Action, State, Param] {
	return fsk.historyKeeper.Items()
}
