package kry

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/fxamacker/cbor/v2"
)

const (
	defaultSkipStackTrace = 3
)

type HistoryItem[Action, State comparable, Param any] struct {
	Action     Action
	From       State
	To         State
	Params     []Param
	Err        error
	StackTrace string
	Reason     string
}

type historyItem[Action, State comparable, Param any] struct {
	*HistoryItem[Action, State, Param]
	Next *historyItem[Action, State, Param]
}

type historyKeeper[Action, State comparable, Param any] struct {
	maxLength  int
	head       *historyItem[Action, State, Param]
	tail       *historyItem[Action, State, Param]
	length     int
	stackTrace bool

	locker sync.Mutex
}

func newHistoryKeeper[Action, State comparable, Param any](size int, stackTrace bool) *historyKeeper[Action, State, Param] {
	return &historyKeeper[Action, State, Param]{
		maxLength:  size,
		head:       nil,
		tail:       nil,
		length:     0,
		stackTrace: stackTrace,
		locker:     sync.Mutex{},
	}
}

func cloneParams[Param any](params ...Param) ([]Param, error) {
	if len(params) == 0 {
		return params, nil
	}

	data, err := cbor.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	var cloned []Param

	if err := cbor.Unmarshal(data, &cloned); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	return cloned, nil
}

func (hk *historyKeeper[Action, State, Param]) Push(action Action, from State, to State, err error, skipStackTrace int, params ...Param) error {
	if hk.maxLength == 0 {
		return nil
	}

	cloneParams, errClone := cloneParams(params...)
	if errClone != nil {
		return fmt.Errorf("failed to clone params: %w", errClone)
	}

	item := &historyItem[Action, State, Param]{
		HistoryItem: &HistoryItem[Action, State, Param]{
			Action: action,
			From:   from,
			To:     to,
			Params: cloneParams,
			Err:    err,
		},
	}

	if hk.stackTrace && err != nil {
		item.Reason = err.Error()
		const depth = 64
		pcs := make([]uintptr, depth)
		// skip 3 frames: runtime.Callers -> push -> Push
		n := runtime.Callers(skipStackTrace, pcs)
		pcs = pcs[:n]

		var b strings.Builder
		frames := runtime.CallersFrames(pcs)
		for {
			frame, ok := frames.Next()
			if !ok {
				break
			}
			fmt.Fprintf(&b, "    %s\n        %s:%d\n", frame.Function, frame.File, frame.Line)
		}

		item.StackTrace = b.String()
	}

	hk.locker.Lock()
	defer hk.locker.Unlock()

	if hk.length == 0 {
		hk.head = item
		hk.tail = item
		hk.length++

		return nil
	}

	if hk.maxLength > 0 && hk.length >= hk.maxLength {
		hk.tail.Next = item
		hk.tail = item
		hk.head = hk.head.Next

		return nil
	}

	hk.length++
	hk.tail.Next = item
	hk.tail = item

	return nil
}

func (hk *historyKeeper[Action, State, Param]) Items() []HistoryItem[Action, State, Param] {
	hk.locker.Lock()
	defer hk.locker.Unlock()

	items := make([]HistoryItem[Action, State, Param], 0, hk.length)

	current := hk.head
	for current != nil {
		items = append(items, *current.HistoryItem)
		current = current.Next
	}

	return items
}

func (hk *historyKeeper[Action, State, Param]) Append(other *historyKeeper[Action, State, Param]) {
	if other.length == 0 {
		return
	}

	hk.locker.Lock()
	defer hk.locker.Unlock()

	if hk.tail == nil {
		hk.head = other.head
		hk.tail = other.tail
		hk.length = other.length

		return
	}

	hk.tail.Next = other.head
	hk.tail = other.tail
	hk.length += other.length

	if hk.maxLength > 0 && hk.length > hk.maxLength {
		excess := hk.length - hk.maxLength
		current := hk.head

		for range excess {
			current = current.Next
		}

		hk.head = current
		hk.length = hk.maxLength
	}
}

// the following methods are added to FSM because they relate to history management

func (fsk *FSM[Action, State, Param]) intermediateKeeper(
	historyKeeper *historyKeeper[Action, State, Param],
	action Action,
	from, to State,
	err error,
	param ...Param,
) (*historyKeeper[Action, State, Param], error) {
	finalKeeper := newHistoryKeeper[Action, State, Param](
		fsk.historyKeeper.maxLength,
		fsk.stackTrace,
	)
	if errHistory := finalKeeper.
		Push(action, from, to, err, defaultSkipStackTrace, param...,
		); errHistory != nil {
		return nil, fmt.Errorf("failed to push history item: %w", errHistory)
	}

	if historyKeeper.length > 0 {
		finalKeeper.Append(historyKeeper)
	}

	return finalKeeper, nil
}

func (fsk *FSM[Action, State, Param]) History() []HistoryItem[Action, State, Param] {
	return fsk.historyKeeper.Items()
}
