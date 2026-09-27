package commands

import (
	"fmt"
	"reflect"

	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-service/models"
)

func getErrorMessage(err error) (message string) {
	getPayload := reflect.ValueOf(err).MethodByName("GetPayload")
	if (getPayload != reflect.Value{}) {
		pt := getPayload.Call([]reflect.Value{})[0]
		if cause, ok := pt.Interface().(*models.Error); ok {
			if cause.Code != nil {
				message = fmt.Sprintf("[%s] ", *cause.Code)
			}
			message = fmt.Sprintf("%s%s", message, swag.StringValue(cause.Reason))
		}
		if cause, ok := pt.Interface().(*models.InfraError); ok {
			if cause.Code != nil {
				message = fmt.Sprintf("[%d] ", *cause.Code)
			}
			message = fmt.Sprintf("%s%s", message, swag.StringValue(cause.Message))
		}
	} else {
		message = err.Error()
	}
	return
}
