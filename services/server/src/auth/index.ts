import { betterAuth } from "better-auth";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import { db, users, sessions, userAccounts, verifications } from "../db";
import {
  sendVerificationEmail,
  sendPasswordResetEmail,
  sendWelcomeEmail,
} from "../lib/email-service";
import { logger } from "../lib/logger";

// Get base URL from environment or default to localhost
const baseURL = process.env.BETTER_AUTH_URL || "http://localhost:3000";
const secret =
  process.env.BETTER_AUTH_SECRET || "your-secret-key-change-in-production";

export const auth = betterAuth({
  database: drizzleAdapter(db, {
    provider: "sqlite", // Using Bun.SQLite
    schema: {
      user: users,
      session: sessions,
      account: userAccounts,
      verification: verifications,
    },
  }),
  emailAndPassword: {
    enabled: true,
    requireEmailVerification: false, // Allow login without email verification
    sendResetPassword: async ({ user, url, token }) => {
      await sendPasswordResetEmail({
        email: user.email,
        resetUrl: url,
        token,
        expiresIn: 60, // 1 hour
      });
    },
  },
  emailVerification: {
    sendVerificationEmail: async ({ user, url, token }) => {
      await sendVerificationEmail({
        email: user.email,
        verificationUrl: url,
        token,
      });
    },
    sendOnSignUp: true, // Automatically send verification email on signup
    sendOnSignIn: false, // Don't resend verification email on sign-in
  },
  baseURL,
  secret,
  // Trusted origins for CSRF protection
  // Allow requests from the main app and example apps
  trustedOrigins: [
    baseURL, // Main app (http://localhost:3000)
    "http://localhost:3001", // Auth example app
    "http://localhost:5173", // Common Vite dev server port
    "http://localhost:5174", // Alternative Vite port
    ...(process.env.BETTER_AUTH_TRUSTED_ORIGINS?.split(",") || []), // Additional origins from env
  ],
  // Rate limiting configuration
  rateLimit: {
    enabled: true,
    window: 15 * 60, // 15 minutes in seconds
    max: 5, // 5 attempts per window
    storage: "database", // Store rate limit data in database
  },
  // OAuth providers can be added later
  // socialProviders: {
  //   google: {
  //     clientId: process.env.GOOGLE_CLIENT_ID!,
  //     clientSecret: process.env.GOOGLE_CLIENT_SECRET!,
  //   },
  //   github: {
  //     clientId: process.env.GITHUB_CLIENT_ID!,
  //     clientSecret: process.env.GITHUB_CLIENT_SECRET!,
  //   },
  // },
  // Database hooks for sending welcome email
  databaseHooks: {
    user: {
      create: {
        after: async ({ user }) => {
          // Send welcome email after user creation (non-blocking)
          // Only send if user has email and name
          if (user?.email && user?.name) {
            try {
              await sendWelcomeEmail(user.email, user.name);
            } catch (error) {
              // Log error but don't fail user creation
              // Email service may not be configured, which is fine
              logger.error("Failed to send welcome email", error, {
                userId: user.id,
                email: user.email,
              });
            }
          }
        },
      },
    },
  },
});

// Export the auth handler for use in Elysia
// The handler is a function that takes a Request and returns a Response
export const authHandler = auth.handler;
