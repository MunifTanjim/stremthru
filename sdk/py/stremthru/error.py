from typing import Any, Literal

from multidict import CIMultiDict

ErrorCode = Literal[
    "BAD_GATEWAY",
    "BAD_REQUEST",
    "CONFLICT",
    "FORBIDDEN",
    "GONE",
    "INTERNAL_SERVER_ERROR",
    "METHOD_NOT_ALLOWED",
    "NOT_FOUND",
    "NOT_IMPLEMENTED",
    "PAYMENT_REQUIRED",
    "PROXY_AUTHENTICATION_REQUIRED",
    "SERVICE_UNAVAILABLE",
    "STORE_LIMIT_EXCEEDED",
    "STORE_MAGNET_INVALID",
    "TOO_MANY_REQUESTS",
    "UNAUTHORIZED",
    "UNAVAILABLE_FOR_LEGAL_REASONS",
    "UNKNOWN",
    "UNPROCESSABLE_ENTITY",
    "UNSUPPORTED_MEDIA_TYPE",
]

ErrorType = Literal["api_error", "store_error", "unknown_error", "upstream_error"]


class StremThruError(Exception):
    body: Any
    code: ErrorCode = "UNKNOWN"
    headers: CIMultiDict
    status_code: int
    type: ErrorType = "unknown_error"

    def __init__(
        self,
        body: Any = "",
        headers: CIMultiDict = CIMultiDict(),
        status_code: int = 500,
    ):
        message = ""
        type: ErrorType = "unknown_error"
        code: ErrorCode = "UNKNOWN"
        if isinstance(body, str):
            message = body
        elif isinstance(body, dict) and "error" in body:
            type = body["error"].get("type", "unknown_error")
            code = body["error"].get("code", "UNKNOWN")
            if "message" in body["error"]:
                message = f"({type}) {body['error'].get('message')}"
            else:
                message = str(body["error"])
        else:
            message = str(body)

        super().__init__(message)

        self.body = body
        self.code = code
        self.headers = headers
        self.status_code = status_code
        self.type = type
