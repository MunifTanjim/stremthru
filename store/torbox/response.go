package torbox

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
)

type ResponseStatus string

const (
	ResponseStatusSuccess ResponseStatus = "success"
	ResponseStatusError   ResponseStatus = "error"
)

type ErrorCode string

const (
	ErrorCodeDatabaseError           ErrorCode = "DATABASE_ERROR"
	ErrorCodeUnknownError            ErrorCode = "UNKNOWN_ERROR"
	ErrorCodeNoAuth                  ErrorCode = "NO_AUTH"
	ErrorCodeBadToken                ErrorCode = "BAD_TOKEN"
	ErrorCodeAuthError               ErrorCode = "AUTH_ERROR"
	ErrorCodeInvalidOption           ErrorCode = "INVALID_OPTION"
	ErrorCodeRedirectError           ErrorCode = "REDIRECT_ERROR"
	ErrorCodeOAuthVerificationError  ErrorCode = "OAUTH_VERIFICATION_ERROR"
	ErrorCodeEndpointNotFound        ErrorCode = "ENDPOINT_NOT_FOUND"
	ErrorCodeItemNotFound            ErrorCode = "ITEM_NOT_FOUND"
	ErrorCodePlanRestrictedFeature   ErrorCode = "PLAN_RESTRICTED_FEATURE"
	ErrorCodeDuplicateItem           ErrorCode = "DUPLICATE_ITEM"
	ErrorCodeBozoRssFeed             ErrorCode = "BOZO_RSS_FEED"
	ErrorCodeSellixError             ErrorCode = "SELLIX_ERROR"
	ErrorCodeTooMuchData             ErrorCode = "TOO_MUCH_DATA"
	ErrorCodeDownloadTooLarge        ErrorCode = "DOWNLOAD_TOO_LARGE"
	ErrorCodeMissingRequiredOption   ErrorCode = "MISSING_REQUIRED_OPTION"
	ErrorCodeTooManyOptions          ErrorCode = "TOO_MANY_OPTIONS"
	ErrorCodeBozoTorrent             ErrorCode = "BOZO_TORRENT"
	ErrorCodeNoServersAvailableError ErrorCode = "NO_SERVERS_AVAILABLE_ERROR"
	ErrorCodeMonthlyLimit            ErrorCode = "MONTHLY_LIMIT"
	ErrorCodeCooldownLimit           ErrorCode = "COOLDOWN_LIMIT"
	ErrorCodeActiveLimit             ErrorCode = "ACTIVE_LIMIT"
	ErrorCodeDownloadServerError     ErrorCode = "DOWNLOAD_SERVER_ERROR"
	ErrorCodeBozoNzb                 ErrorCode = "BOZO_NZB"
	ErrorCodeSearchError             ErrorCode = "SEARCH_ERROR"
	ErrorCodeInvalidDevice           ErrorCode = "INVALID_DEVICE"
	ErrorCodeDiffIssue               ErrorCode = "DIFF_ISSUE"
	ErrorCodeLinkOffline             ErrorCode = "LINK_OFFLINE"
	ErrorCodeVendorDisabled          ErrorCode = "VENDOR_DISABLED"
)

type ResponseError struct {
	Detail string    `json:"detail"`
	Err    ErrorCode `json:"error"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type Response[T any] struct {
	Success bool      `json:"success"`
	Data    T         `json:"data,omitempty"`
	Detail  string    `json:"detail"`
	Error   ErrorCode `json:"error,omitempty"`
}

type ResponseEnvelop interface {
	IsSuccess() bool
	GetError() *ResponseError
}

func (r Response[any]) IsSuccess() bool {
	return r.Success
}

func (r Response[any]) GetError() *ResponseError {
	if r.IsSuccess() {
		return nil
	}
	return &ResponseError{
		Err:    r.Error,
		Detail: r.Detail,
	}
}

type APIResponse[T any] struct {
	Header     http.Header
	StatusCode int
	Data       T
	Detail     string
}

func newAPIResponse[T any](res *http.Response, data T, detail string) APIResponse[T] {
	return APIResponse[T]{
		Header:     res.Header,
		StatusCode: res.StatusCode,
		Data:       data,
		Detail:     detail,
	}
}

func extractResponseError(statusCode int, body []byte, v ResponseEnvelop) error {
	if !v.IsSuccess() {
		return v.GetError()
	}
	if statusCode >= http.StatusBadRequest {
		return errors.New(string(body))
	}
	return nil
}

func processResponseBody(res *http.Response, err error, v ResponseEnvelop) error {
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		return err
	}

	err = core.UnmarshalJSON(res.StatusCode, body, v)
	if err != nil {
		return err
	}

	return extractResponseError(res.StatusCode, body, v)
}
