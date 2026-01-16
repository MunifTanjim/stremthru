package znab

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
)

type Error struct {
	XMLName     xml.Name `xml:"error"`
	Code        int      `xml:"code,attr"`
	Description string   `xml:"description,attr"`
}

func (e Error) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Description)
}

type jsonError struct {
	Error struct {
		Attributes struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
		} `json:"@attributes"`
	} `json:"error"`
}

func (e Error) MarshalJSON() ([]byte, error) {
	jErr := jsonError{}
	jErr.Error.Attributes.Code = e.Code
	jErr.Error.Attributes.Description = e.Description
	return json.Marshal(jErr)
}

var (
	// Account/user credentials specific error codes

	ErrorIncorrectUserCreds     = Error{Code: 100, Description: "Incorrect user credentials"}
	ErrorAccountSuspended       = Error{Code: 101, Description: "Account suspended"}
	ErrorInsufficientPrivs      = Error{Code: 102, Description: "Insufficient privileges/not authorized"}
	ErrorRegistrationDenied     = Error{Code: 103, Description: "Registration denied"}
	ErrorRegistrationsAreClosed = Error{Code: 104, Description: "Registrations are closed"}
	ErrorEmailAddressTaken      = Error{Code: 105, Description: "Invalid registration (Email Address Taken)"}
	ErrorEmailAddressBadFormat  = Error{Code: 106, Description: "Invalid registration (Email Address Bad Format)"}
	ErrorRegistrationFailed     = Error{Code: 107, Description: "Registration Failed (Data error)"}

	// API call specific error codes

	ErrorMissingParameter = func(desc string) error {
		err := Error{Code: 200, Description: "Missing parameter"}
		if desc != "" {
			err.Description += ": " + desc
		}
		return err
	}
	ErrorIncorrectParameter = func(desc string) error {
		err := Error{Code: 201, Description: "Incorrect parameter"}
		if desc != "" {
			err.Description += ": " + desc
		}
		return err
	}
	ErrorNoSuchFunction       = Error{Code: 202, Description: "No such function. (Function not defined in this specification)."}
	ErrorFunctionNotAvailable = Error{Code: 203, Description: "Function not available. (Optional function is not implemented)."}

	// Content specific error codes

	ErrorNoSuchItem = Error{Code: 300, Description: "No such item."}

	ErrorRequestLimitReached  = Error{Code: 500, Description: "Request limit reached"}
	ErrorDownloadLimitReached = Error{Code: 501, Description: "Download limit reached"}

	// Other error codes

	ErrorUnknownError = func(desc string) error {
		err := Error{Code: 900, Description: "Unknown error"}
		if desc != "" {
			err.Description += ": " + desc
		}
		return err
	}

	ErrorAPIDisabled = Error{Code: 910, Description: "API Disabled"}
)
