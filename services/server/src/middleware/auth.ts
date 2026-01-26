import { Elysia, type Context } from "elysia";
import { auth } from "../auth";

// User type
export type AuthenticatedUser = {
  id: string;
  email: string;
  name: string;
  emailVerified: boolean;
  image?: string | null;
};

// Extend Elysia context with user

/**
 * Authentication resolver function
 * Used with .resolve() for type-safe context extension
 * Runs at beforeHandle lifecycle (after validation)
 * Returns early with 401 status if authentication fails
 */
export const authResolver = async ({ request, headers, status }: Context) => {
  try {
    // Better Auth getSession accepts headers directly
    // Use headers from context (available in resolve) or fallback to request.headers
    const sessionHeaders = headers || request.headers;
    const session = await auth.api.getSession({
      headers: sessionHeaders,
    });

    if (!session?.user) {
      return status(401, {
        error: {
          message: "No valid session found",
          code: "UNAUTHORIZED",
        },
      });
    }

    return {
      user: {
        id: session.user.id,
        email: session.user.email,
        name: session.user.name,
        emailVerified: session.user.emailVerified,
        image: session.user.image,
      } satisfies AuthenticatedUser,
    };
  } catch (error) {
    // Log error for debugging (only in development)
    if (process.env.NODE_ENV !== "production") {
      console.error("Auth resolver error:", error);
    }
    return status(401, {
      error: {
        message: "Authentication required",
        code: "UNAUTHORIZED",
      },
    });
  }
};

/**
 * Authentication middleware plugin (for backward compatibility)
 * Uses authResolver internally
 */
export const authMiddleware = new Elysia({ name: "auth" }).resolve(
  authResolver,
);

/**
 * Optional authentication middleware using resolve
 * Similar to authMiddleware but doesn't require authentication
 * User will be undefined if not authenticated, but request continues
 * Uses resolve for type safety (runs after validation)
 */
export const optionalAuthMiddleware = new Elysia({
  name: "optionalAuth",
}).resolve(async ({ request, headers }) => {
  try {
    const sessionHeaders = headers || request.headers;
    const session = await auth.api.getSession({ headers: sessionHeaders });
    return {
      user: session?.user
        ? ({
            id: session.user.id,
            email: session.user.email,
            name: session.user.name,
            emailVerified: session.user.emailVerified,
            image: session.user.image,
          } satisfies AuthenticatedUser)
        : undefined,
    };
  } catch {
    // Silently fail and continue without user
    return { user: undefined };
  }
});
