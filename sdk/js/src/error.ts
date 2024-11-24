export type ErrorCode =
  | "BAD_GATEWAY"
  | "BAD_REQUEST"
  | "CONFLICT"
  | "FORBIDDEN"
  | "GONE"
  | "INTERNAL_SERVER_ERROR"
  | "MAGNET_INVALID"
  | "METHOD_NOT_ALLOWED"
  | "NOT_FOUND"
  | "NOT_IMPLEMENTED"
  | "PAYMENT_REQUIRED"
  | "PROXY_AUTHENTICATION_REQUIRED"
  | "SERVICE_UNAVAILABLE"
  | "STORE_LIMIT_EXCEEDED"
  | "TOO_MANY_REQUESTS"
  | "UNAUTHORIZED"
  | "UNAVAILABLE_FOR_LEGAL_REASONS"
  | "UNKNOWN"
  | "UNPROCESSABLE_ENTITY"
  | "UNSUPPORTED_MEDIA_TYPE";

export type ErrorType =
  | "api_error"
  | "store_error"
  | "unknown_error"
  | "upstream_error";

export class StremThruError extends Error {
  body?: unknown;
  code?: ErrorCode = "UNKNOWN";
  headers: Record<string, string>;
  statusCode: number;
  statusText: string;
  type?: ErrorType = "unknown_error";

  constructor(
    message: string,
    options: Pick<
      StremThruError,
      "body" | "code" | "headers" | "statusCode" | "statusText" | "type"
    > & { cause?: unknown },
  ) {
    super(message);

    // Maintains proper stack trace for where our error was thrown (only available on V8)
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, this.constructor);
    }

    if (options?.cause) {
      this.cause = options.cause;
      delete options.cause;
    }

    if (options.body) {
      this.body = options.body;
    }

    this.headers = options.headers;
    this.statusCode = options.statusCode;
    this.statusText = options.statusText;

    if (options.type) {
      this.type = options.type;
    }
    if (options.code) {
      this.code = options.code;
    }
  }
}
