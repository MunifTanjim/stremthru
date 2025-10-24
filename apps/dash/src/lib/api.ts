type Response<
  Data extends null | Record<string, unknown>,
  Params extends null | Record<string, string> = Record<string, string>,
  Meta extends Record<string, unknown> = Record<string, unknown>,
> = {
  data: Data;
  meta: Meta;
  params: Params;
};

export class APIError extends Error {
  error: {
    code: number;
    errors: Array<{
      domain?: string;
      extendedHelp?: string;
      location?: string;
      locationType?: string;
      message: string;
      reason?: string;
      sendReport?: string;
    }>;
    message: string;
  };
  status: number;
  constructor(status: APIError["status"], error: APIError["error"]) {
    super(error.message);
    this.status = status;
    this.error = error;
  }
}

export async function api<
  Data extends null | Record<string, unknown>,
  Params extends null | Record<string, string> = null,
  Meta extends Record<string, unknown> = Record<string, unknown>,
>(
  endpoint: string,
  {
    body,
    ...options
  }: Omit<RequestInit, "body"> & {
    body?: FormData | Record<string, unknown> | string;
  } = {},
): Promise<Response<Data, Params, Meta & { status: number }>> {
  const url = `/dash/api${endpoint}`;

  const request: RequestInit = {
    credentials: "include",
    method: "GET",
    ...options,
  };
  request.method = request.method!.toUpperCase();
  const headers = new Headers({
    ...request.headers,
  });

  if (body instanceof FormData) {
    request.body = body;
  } else if (!["string", "undefined"].includes(typeof body)) {
    headers.set("content-type", "application/json");
    request.body = JSON.stringify(body);
  }

  request.headers = headers;

  const res = await fetch(url, request);

  const contentType = res.headers.get("content-type");

  if (!/application\/json/.test(contentType ?? "")) {
    if (res.status !== 204) {
      throw new Error(`Unsupported Content-Type: ${contentType}`);
    }
  }

  const { data, error, params, ...meta } =
    res.status === 204
      ? { data: null, error: null, params: null }
      : await res.json();

  if (error) {
    throw new APIError(res.status, error);
  }

  return {
    data,
    meta: {
      ...meta,
      status: res.status,
    },
    params,
  };
}

export function extractErrorMessages(err: unknown) {
  const errors: string[] = [];

  if (!err) {
    return errors;
  }

  if (err instanceof APIError) {
    const error = err.error;

    if (error.message !== error.errors[0]?.message) {
      errors.push(error.message);
    }

    for (const { location, message } of error.errors) {
      errors.push(location ? `${location}: ${message}` : message);
    }
  } else if (err instanceof Error) {
    errors.push(err.message);
  } else {
    errors.push(`Something went wrong!`);
  }

  console.error(err);

  return errors;
}
