import { Elysia } from "elysia";

export class AppError extends Error {
  constructor(
    public message: string,
    public statusCode: number = 500,
    public code?: string,
  ) {
    super(message);
    this.name = "AppError";
  }
}

export class ValidationError extends AppError {
  constructor(
    message: string,
    public details?: unknown,
  ) {
    super(message, 400, "VALIDATION_ERROR");
    this.name = "ValidationError";
  }
}

export class NotFoundError extends AppError {
  constructor(resource: string, id?: string) {
    super(
      id ? `${resource} with id ${id} not found` : `${resource} not found`,
      404,
      "NOT_FOUND",
    );
    this.name = "NotFoundError";
  }
}

export class UnauthorizedError extends AppError {
  constructor(message: string = "Unauthorized") {
    super(message, 401, "UNAUTHORIZED");
    this.name = "UnauthorizedError";
  }
}

export class ForbiddenError extends AppError {
  constructor(message: string = "Forbidden") {
    super(message, 403, "FORBIDDEN");
    this.name = "ForbiddenError";
  }
}

export function formatErrorResponse(error: unknown) {
  if (error instanceof AppError) {
    return {
      error: {
        message: error.message,
        code: error.code,
        statusCode: error.statusCode,
        ...(error instanceof ValidationError && { details: error.details }),
      },
    };
  }

  // Handle Zod validation errors
  if (error && typeof error === "object" && "issues" in error) {
    return {
      error: {
        message: "Validation failed",
        code: "VALIDATION_ERROR",
        statusCode: 400,
        details: error,
      },
    };
  }

  // Handle database errors
  if (error && typeof error === "object" && "code" in error) {
    const dbError = error as { code: string; message: string };
    return {
      error: {
        message: dbError.message || "Database error",
        code: dbError.code || "DATABASE_ERROR",
        statusCode: 500,
      },
    };
  }

  // Generic error
  return {
    error: {
      message: error instanceof Error ? error.message : "Internal server error",
      code: "INTERNAL_ERROR",
      statusCode: 500,
    },
  };
}

export const errorHandler = new Elysia().onError(({ code, error, set }) => {
  const response = formatErrorResponse(error);
  set.status = response.error.statusCode;
  console.error(error);
  return response;
});
