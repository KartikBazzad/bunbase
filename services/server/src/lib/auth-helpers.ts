import type { AuthenticatedUser } from "../middleware/auth";

/**
 * Type guard to ensure user is authenticated
 * 
 * Note: After authMiddleware, if we reach the handler, user should always be present
 * because authMiddleware returns early with 401 if authentication fails.
 * This type guard is for TypeScript's benefit to narrow the type.
 */
export function requireAuth(user: AuthenticatedUser | undefined): asserts user is AuthenticatedUser {
  if (!user) {
    // This should never happen if authMiddleware is properly applied
    // but we check for type safety
    throw new Error("User not authenticated - this should not happen after authMiddleware");
  }
}
