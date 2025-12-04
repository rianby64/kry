package kry

import (
	"reflect"
)

func (fsk *FSM[Action, State, Param]) decorateWithExpectApply(callbacks callbacks[Action, State, Param]) expectFailed {
	var (
		expectToCallEnterNoParams []handlerNoParams[Action, State, Param]
		expectToCallEnter         []handler[Action, State, Param]
		expectToCallEnterVariadic []handlerVariadic[Action, State, Param]

		expectFailed expectFailed
	)

	if fsk.decoratorApply != nil {
		if len(fsk.decoratorApply.expectToCallEnter) > 0 {
			expectedEnterFound := false
			expectToCallEnter = fsk.decoratorApply.expectToCallEnter
			fsk.decoratorApply.expectToCallEnter = nil

			pointerToEnter := reflect.ValueOf(callbacks.Enter).Pointer()
			for _, expectedHandler := range expectToCallEnter {
				if pointerToEnter == reflect.ValueOf(expectedHandler).Pointer() {
					expectedEnterFound = true

					break
				}
			}

			expectFailed.Enter = !expectedEnterFound
		}

		if len(fsk.decoratorApply.expectToCallEnter) > 0 {
			expectedEnterNoParamsFound := false
			expectToCallEnterNoParams = fsk.decoratorApply.expectToCallEnterNoParams
			fsk.decoratorApply.expectToCallEnterNoParams = nil

			pointerToEnterNoParams := reflect.ValueOf(callbacks.Enter).Pointer()
			for _, expectedHandler := range expectToCallEnterNoParams {
				if pointerToEnterNoParams == reflect.ValueOf(expectedHandler).Pointer() {
					expectedEnterNoParamsFound = true

					break
				}
			}

			expectFailed.EnterNoParams = !expectedEnterNoParamsFound
		}

		if len(fsk.decoratorApply.expectToCallEnter) > 0 {
			expectedEnterVariadicFound := false
			expectToCallEnterVariadic = fsk.decoratorApply.expectToCallEnterVariadic
			fsk.decoratorApply.expectToCallEnterVariadic = nil

			pointerToEnterVariadic := reflect.ValueOf(callbacks.EnterVariadic).Pointer()
			for _, expectedHandler := range expectToCallEnterVariadic {
				if pointerToEnterVariadic == reflect.ValueOf(expectedHandler).Pointer() {
					expectedEnterVariadicFound = true

					break
				}
			}

			expectFailed.EnterVariadic = !expectedEnterVariadicFound
		}
	}

	return expectFailed
}
