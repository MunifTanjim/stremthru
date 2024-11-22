export class StremThruError extends Error {
  body?: unknown;
  code?: string;
  headers: Record<string, string>;
  statusCode: number;
  statusText: string;
  type?: "api_error" | "store_error" | "unknown_error" | "upstream_error" =
    "unknown_error";

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
