import { betterAuth } from "better-auth";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import { db, users, sessions, userAccounts, verifications } from "../db";
import {
  sendVerificationEmail,
  sendPasswordResetEmail,
  sendWelcomeEmail,
} from "../lib/email-service";

// Get base URL from environment or default to localhost
const baseURL = process.env.BETTER_AUTH_URL || "http://localhost:3000";
const secret = process.env.BETTER_AUTH_SECRET || "your-secret-key-change-in-production";

export const auth = betterAuth({
  database: drizzleAdapter(db, {
    provider: "pg", // PGLite is PostgreSQL-compatible
    schema: {
      user: users,
      session: sessions,
      account: userAccounts,
      verification: verifications,
    },
  }),
  emailAndPassword: {
    enabled: true,
    requireEmailVerification: true, // Enable email verification
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
  },
  baseURL,
  secret,
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
          // Send welcome email after user creation
          try {
            await sendWelcomeEmail(user.email, user.name);
          } catch (error) {
            // Log error but don't fail user creation
            console.error("Failed to send welcome email:", error);
          }
        },
      },
    },
  },
});

// Export the auth handler for use in Elysia
// The handler is a function that takes a Request and returns a Response
export const authHandler = auth.handler;
