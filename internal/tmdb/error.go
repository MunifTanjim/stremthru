package tmdb

type ErrorCode int

const (
	ErrorCodeSuccess                 ErrorCode = 1  //	Success.
	ErrorCodeInvalidService          ErrorCode = 2  //	Invalid service: this service does not exist.
	ErrorCodeNoPermission            ErrorCode = 3  //	Authentication failed: You do not have permissions to access the service.
	ErrorCodeInvalidFormat           ErrorCode = 4  //	Invalid format: This service doesn't exist in that format.
	ErrorCodeInvalidParameters       ErrorCode = 5  //	Invalid parameters: Your request parameters are incorrect.
	ErrorCodeInvalidPreRequisiteId   ErrorCode = 6  //	Invalid id: The pre-requisite id is invalid or not found.
	ErrorCodeInvalidAPIKey           ErrorCode = 7  //	Invalid API key: You must be granted a valid key.
	ErrorCodeDuplicateEntry          ErrorCode = 8  //	Duplicate entry: The data you tried to submit already exists.
	ErrorCodeServiceOffline          ErrorCode = 9  //	Service offline: This service is temporarily offline, try again later.
	ErrorCodeSuspendedAPIKey         ErrorCode = 10 //	Suspended API key: Access to your account has been suspended, contact TMDB.
	ErrorCodeInternalError           ErrorCode = 11 //	Internal error: Something went wrong, contact TMDB.
	ErrorCodeUpdatedSuccessfully     ErrorCode = 12 //	The item/record was updated successfully.
	ErrorCodeDeletedSuccessfully     ErrorCode = 13 //	The item/record was deleted successfully.
	ErrorCodeAuthenticationFailed    ErrorCode = 14 //	Authentication failed.
	ErrorCodeFailed                  ErrorCode = 15 //	Failed.
	ErrorCodeDeviceDenied            ErrorCode = 16 //	Device denied.
	ErrorCodeSessionDenied           ErrorCode = 17 //	Session denied.
	ErrorCodeValidationFailed        ErrorCode = 18 //	Validation failed.
	ErrorCodeInvalidAcceptHeader     ErrorCode = 19 //	Invalid accept header.
	ErrorCodeInvalidDateRange        ErrorCode = 20 //	Invalid date range: Should be a range no longer than 14 days.
	ErrorCodeEntryNotFound           ErrorCode = 21 //	Entry not found: The item you are trying to edit cannot be found.
	ErrorCodeInvalidPage             ErrorCode = 22 //	Invalid page: Pages start at 1 and max at 500. They are expected to be an integer.
	ErrorCodeInvalidDate             ErrorCode = 23 //	Invalid date: Format needs to be YYYY-MM-DD.
	ErrorCodeRequestTimedOut         ErrorCode = 24 //	Your request to the backend server timed out. Try again.
	ErrorCodeRequestLimitExceeded    ErrorCode = 25 //	Your request count (#) is over the allowed limit of (40).
	ErrorCodeUserPassMissing         ErrorCode = 26 //	You must provide a username and password.
	ErrorCodeTooManyAppendToResponse ErrorCode = 27 //	Too many append to response objects: The maximum number of remote calls is 20.
	ErrorCodeInvalidTimezone         ErrorCode = 28 //	Invalid timezone: Please consult the documentation for a valid timezone.
	ErrorCodeConfirmationNeeded      ErrorCode = 29 //	You must confirm this action: Please provide a confirm=true parameter.
	ErrorCodeInvalidUserPass         ErrorCode = 30 //	Invalid username and/or password: You did not provide a valid login.
	ErrorCodeAccountDisabled         ErrorCode = 31 //	Account disabled: Your account is no longer active. Contact TMDB if this is an error.
	ErrorCodeEmailNotVerified        ErrorCode = 32 //	Email not verified: Your email address has not been verified.
	ErrorCodeInvalidRequestToken     ErrorCode = 33 //	Invalid request token: The request token is either expired or invalid.
	ErrorCodeNotFound                ErrorCode = 34 //	The resource you requested could not be found.
	ErrorCodeInvalidToken            ErrorCode = 35 //	Invalid token.
	ErrorCodeNotGrantedWrite         ErrorCode = 36 //	This token hasn't been granted write permission by the user.
	ErrorCodeSessionNotFound         ErrorCode = 37 //	The requested session could not be found.
	ErrorCodeEditForbidden           ErrorCode = 38 //	You don't have permission to edit this resource.
	ErrorCodePrivateResource         ErrorCode = 39 //	This resource is private.
	ErrorCodeNothingToUpdate         ErrorCode = 40 //	Nothing to update.
	ErrorCodeNotApproved             ErrorCode = 41 //	This request token hasn't been approved by the user.
	ErrorCodeMethodNotAllowed        ErrorCode = 42 //	This request method is not supported for this resource.
	ErrorCodeConnectionError         ErrorCode = 43 //	Couldn't connect to the backend server.
	ErrorCodeInvalidId               ErrorCode = 44 //	The ID is invalid.
	ErrorCodeSuspendedUser           ErrorCode = 45 //	This user has been suspended.
	ErrorCodeMaintenance             ErrorCode = 46 //	The API is undergoing maintenance. Try again later.
	ErrorCodeInvalidInput            ErrorCode = 47 //	The input is not valid.
)

var httpStatusByErrorCode = map[int]int{
	1:  200,
	2:  501,
	3:  401,
	4:  405,
	5:  422,
	6:  404,
	7:  401,
	8:  403,
	9:  503,
	10: 401,
	11: 500,
	12: 201,
	13: 200,
	14: 401,
	15: 500,
	16: 401,
	17: 401,
	18: 400,
	19: 406,
	20: 422,
	21: 200,
	22: 400,
	23: 400,
	24: 504,
	25: 429,
	26: 400,
	27: 400,
	28: 400,
	29: 400,
	30: 401,
	31: 401,
	32: 401,
	33: 401,
	34: 404,
	35: 401,
	36: 401,
	37: 404,
	38: 401,
	39: 401,
	40: 200,
	41: 422,
	42: 405,
	43: 502,
	44: 500,
	45: 403,
	46: 503,
	47: 400,
}
