package util

import "github.com/stretchr/testify/mock"

func GetCallsMatchPartialArguments(mockObject *mock.Mock, methodName string, arguments ...interface{}) map[int]*mock.Call {
	var expectedCalls = make(map[int]*mock.Call)

	for index, call := range mockObject.ExpectedCalls {
		if call.Method == methodName {
			var callArguments mock.Arguments

			if len(arguments) == len(call.Arguments) {
				callArguments = call.Arguments
			} else {
				callArguments = call.Arguments[:len(arguments)]
			}

			_, diffCount := callArguments.Diff(arguments)

			if diffCount == 0 {
				expectedCalls[index] = call
			}
		}
	}

	return expectedCalls
}

func GetExpectedCall(mockObject *mock.Mock, methodName string, arguments ...interface{}) (int, *mock.Call) {
	var expectedCall *mock.Call

	for i, call := range mockObject.ExpectedCalls {
		if call.Method == methodName {
			_, diffCount := call.Arguments.Diff(arguments)
			if diffCount == 0 {
				expectedCall = call
				if call.Repeatability > -1 {
					return i, call
				}
			}
		}
	}

	return -1, expectedCall
}

func IncrementCall(mockObject *mock.Mock, index int) {
	call := mockObject.ExpectedCalls[index]
	DeleteExpectedCall(mockObject, index)
	mockObject.On(call.Method, call.Arguments...).Return(call.ReturnArguments...).Times(call.Repeatability + 1)
}

// deleteExpectedCall allows deleting call expectations from a mock according to method name and arguments
func DeleteExpectedMethod(mockObject *mock.Mock, methodName string, arguments ...interface{}) bool {
	index, _ := GetExpectedCall(mockObject, methodName, arguments...)

	if index >= 0 {
		DeleteExpectedCall(mockObject, index)
		return true
	}

	return false
}

func DeleteExpectedCall(mockObject *mock.Mock, index int) {
	mockObject.ExpectedCalls = append(
		mockObject.ExpectedCalls[:index],
		mockObject.ExpectedCalls[index+1:]...,
	)
}
