/**
 * BunStore Utility Functions
 */

import type { WhereFilterOp } from "./types";

/**
 * Convert BunStore query operator to backend filter format
 */
export function convertWhereOperator(
  op: WhereFilterOp,
  value: any,
): Record<string, any> {
  switch (op) {
    case "==":
      return value;
    case "!=":
      return { $ne: value };
    case "<":
      return { $lt: value };
    case "<=":
      return { $lte: value };
    case ">":
      return { $gt: value };
    case ">=":
      return { $gte: value };
    case "in":
      return { $in: Array.isArray(value) ? value : [value] };
    case "array-contains":
      // For array-contains, we'll need special handling
      // For now, map to a simple equality check
      return { $contains: value };
    case "array-contains-any":
      return { $containsAny: Array.isArray(value) ? value : [value] };
    case "not-in":
      return { $nin: Array.isArray(value) ? value : [value] };
    default:
      return value;
  }
}

/**
 * Build filter object from query constraints
 */
export function buildFilter(
  constraints: Array<{ field: string; op: WhereFilterOp; value: any }>,
): Record<string, any> {
  const filter: Record<string, any> = {};

  for (const constraint of constraints) {
    const { field, op, value } = constraint;
    const filterValue = convertWhereOperator(op, value);

    if (op === "==") {
      filter[field] = value;
    } else {
      filter[field] = filterValue;
    }
  }

  return filter;
}

/**
 * Build sort object from order by constraints
 */
export function buildSort(
  constraints: Array<{ field: string; direction: "asc" | "desc" }>,
): Record<string, "asc" | "desc"> {
  const sort: Record<string, "asc" | "desc"> = {};

  for (const constraint of constraints) {
    sort[constraint.field] = constraint.direction;
  }

  return sort;
}

/**
 * Check if a value is a FieldValue placeholder
 */
export function isFieldValue(value: any): boolean {
  return (
    value &&
    typeof value === "object" &&
    "_type" in value &&
    typeof value._type === "string"
  );
}

/**
 * Serialize FieldValue for API requests
 */
export function serializeFieldValue(value: any): any {
  if (isFieldValue(value)) {
    return {
      __fieldValue: value._type,
      __value: value._value,
    };
  }
  if (Array.isArray(value)) {
    return value.map(serializeFieldValue);
  }
  if (value && typeof value === "object") {
    const serialized: Record<string, any> = {};
    for (const [key, val] of Object.entries(value)) {
      serialized[key] = serializeFieldValue(val);
    }
    return serialized;
  }
  return value;
}

/**
 * Create BunStore error
 */
export function createBunStoreError(
  code: string,
  message: string,
): Error & { code: string } {
  const error = new Error(message) as Error & { code: string };
  error.code = code;
  error.name = "BunStoreError";
  return error;
}

// Alias for backward compatibility
export const createFirestoreError = createBunStoreError;

/**
 * Map backend error to BunStore-like error
 */
export function mapBackendError(error: any): Error & { code: string } {
  if (error.code) {
    // Already has a code
    return error as Error & { code: string };
  }

  // Map HTTP status codes to BunStore error codes
  if (error.status === 404) {
    return createBunStoreError("not-found", "Document not found");
  }
  if (error.status === 403) {
    return createBunStoreError("permission-denied", "Permission denied");
  }
  if (error.status === 400) {
    return createBunStoreError(
      "invalid-argument",
      error.message || "Invalid argument",
    );
  }
  if (error.status === 409) {
    return createBunStoreError("already-exists", "Document already exists");
  }
  if (error.status >= 500) {
    return createBunStoreError("internal", "Internal server error");
  }

  return createBunStoreError("unknown", error.message || "Unknown error");
}
