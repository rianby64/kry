package kry

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
)

const (
	defaultSkipStackTrace = 3
)

type HistoryItem[State comparable, Param any] struct {
	// Below, these are the inputs

	Name   string
	From   State
	To     State
	Params []Param

	// and, these are the outputs...

	Err        error
	StackTrace string
	Reason     string
	Ignored    bool
	Forced     bool
	ForcedTo   *State

	ExpectFailed bool
}

// HasViolation reports whether an escape hatch was used for this transition.
func (h HistoryItem[State, Param]) HasViolation() bool {
	return h.Ignored || h.Forced
}

type historyItem[State comparable, Param any] struct {
	*HistoryItem[State, Param]
	Next *historyItem[State, Param]
}

type historyKeeper[State comparable, Param any] struct {
	maxLength  int
	head       *historyItem[State, Param]
	tail       *historyItem[State, Param]
	length     int
	stackTrace bool

	locker sync.Mutex

	cloneHandler func(params ...Param) ([]Param, error)
}

func newHistoryKeeper[State comparable, Param any](
	size int,
	stackTrace bool,
	cloneHandler CloneHandler[Param],
) *historyKeeper[State, Param] {
	return &historyKeeper[State, Param]{
		maxLength:  size,
		head:       nil,
		tail:       nil,
		length:     0,
		stackTrace: stackTrace,
		locker:     sync.Mutex{},

		cloneHandler: cloneHandler,
	}
}

func newHistoryItem[State comparable, Param any](
	name string,
	from State,
	to State,
	err error,
	ignored bool,
	expectFailed bool,
	forced bool,
	forcedTo *State,
	params ...Param,
) *historyItem[State, Param] {
	return &historyItem[State, Param]{
		HistoryItem: &HistoryItem[State, Param]{
			Name:         name,
			From:         from,
			To:           to,
			Err:          err,
			Ignored:      ignored,
			ExpectFailed: expectFailed,
			Forced:       forced,
			ForcedTo:     forcedTo,
			Params:       params,
		},
	}
}

func cloneHandler[Param any](params ...Param) ([]Param, error) {
	if len(params) == 0 {
		return params, nil
	}

	cloned := make([]Param, len(params))

	copy(cloned, params)

	return cloned, nil
}

// Push records a history item. Its signature is stable — existing callers are unaffected.
func (hk *historyKeeper[State, Param]) Push(
	name string, from State, to State,
	err error, skipStackTrace int, ignored bool, expectFailed bool,
	params ...Param,
) error {
	return hk.push(name, from, to, err, skipStackTrace, ignored, expectFailed, false, nil, params...)
}

func (hk *historyKeeper[State, Param]) push(
	name string, from State, to State,
	err error, skipStackTrace int, ignored bool, expectFailed bool,
	forced bool, forcedTo *State,
	params ...Param,
) error {
	if hk.maxLength == 0 {
		return nil
	}

	cloneParams, errClone := hk.cloneHandler(params...)
	if errClone != nil {
		return fmt.Errorf("failed to clone params: %w", errClone)
	}

	item := newHistoryItem(name, from, to, err, ignored, expectFailed, forced, forcedTo, cloneParams...)

	if hk.stackTrace && err != nil {
		item.Reason = err.Error()
		const depth = 64
		pcs := make([]uintptr, depth)
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

func (hk *historyKeeper[State, Param]) Items() []HistoryItem[State, Param] {
	hk.locker.Lock()
	defer hk.locker.Unlock()

	items := make([]HistoryItem[State, Param], 0, hk.length)

	current := hk.head
	for current != nil {
		items = append(items, *current.HistoryItem)
		current = current.Next
	}

	return items
}

func (hk *historyKeeper[State, Param]) Append(other *historyKeeper[State, Param]) {
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

func (fsk *FSM[State, Param]) intermediateKeeper(
	historyKeeper *historyKeeper[State, Param],
	name string,
	from, to State,
	err error,
	ignored bool,
	expectFailed bool,
	forced bool,
	forcedTo *State,
	param ...Param,
) (*historyKeeper[State, Param], error) {
	finalKeeper := newHistoryKeeper[State](
		fsk.historyKeeper.maxLength,
		fsk.stackTrace,
		fsk.cloneHandler,
	)

	errHistory := finalKeeper.push(
		name, from, to,
		err, defaultSkipStackTrace, ignored, expectFailed,
		forced, forcedTo,
		param...,
	)
	if errHistory != nil {
		return nil, fmt.Errorf("failed to push history item: %w", errHistory)
	}

	if historyKeeper.length > 0 {
		finalKeeper.Append(historyKeeper)
	}

	return finalKeeper, nil
}

func (fsk *FSM[State, Param]) History() []HistoryItem[State, Param] {
	return fsk.historyKeeper.Items()
}
