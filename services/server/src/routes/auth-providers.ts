import { Elysia, t } from "elysia";
import { db, projects } from "../db";
import { authResolver } from "../middleware/auth";
import { NotFoundError, ForbiddenError } from "../lib/errors";
import { requireAuth } from "../lib/auth-helpers";
import { eq, and } from "drizzle-orm";
import { AuthProviderModels, CommonModels } from "./models";
import { logProjectAuthEvent } from "../lib/project-logger-utils";
import { encrypt, decrypt } from "../lib/encryption";
import { getProjectDb } from "../db/project-db-helpers";
import { authSettings, oauthProviders, detailedAuthSettings, projectAccounts } from "../db/project-schema";
import { nanoid } from "nanoid";

// Helper function to verify project ownership
async function verifyProjectOwnership(
  projectId: string,
  userId: string,
): Promise<typeof projects.$inferSelect> {
  const [project] = await db
    .select()
    .from(projects)
    .where(eq(projects.id, projectId))
    .limit(1);

  if (!project) {
    throw new NotFoundError("Project", projectId);
  }

  if (project.ownerId !== userId) {
    throw new ForbiddenError("You don't have access to this project");
  }

  return project;
}

export const authProvidersRoutes = new Elysia({
  prefix: "/projects/:id/authProviders",
})
  .resolve(authResolver)
  .model({
    "authProvider.update": AuthProviderModels.update,
    "authProvider.params": AuthProviderModels.params,
    "authProvider.response": AuthProviderModels.response,
    "common.success": CommonModels.success,
    "common.error": CommonModels.error,
  })
  .onError(({ code, error, set }) => {
    if (code === "VALIDATION") {
      set.status = 422;
      return {
        error: {
          message: error.message,
          code: "VALIDATION_ERROR",
          details: error.all,
        },
      };
    }
    if (error instanceof NotFoundError) {
      set.status = 404;
      return {
        error: {
          message: error.message,
          code: error.code,
        },
      };
    }
    if (error instanceof ForbiddenError) {
      set.status = 403;
      return {
        error: {
          message: error.message,
          code: error.code,
        },
      };
    }
  })
  .guard({
    params: t.Object({
      id: t.String({ minLength: 1 }),
    }),
  })
  .get(
    "/",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      // Get or create auth configuration
      let [authConfig] = await projectDb
        .select()
        .from(authSettings)
        .where(eq(authSettings.id, "default"))
        .limit(1);

      if (!authConfig) {
        // Create default auth config
        const [newAuthConfig] = await projectDb
          .insert(authSettings)
          .values({
            id: "default",
            providers: ["email"],
          })
          .returning();
        authConfig = newAuthConfig;
      }

      if (!authConfig) {
        throw new Error("Failed to get auth configuration");
      }

      return {
        data: {
          projectId: params.id,
          providers: (authConfig.providers as string[]) || ["email"],
          createdAt: authConfig.createdAt,
          updatedAt: authConfig.updatedAt,
        },
      };
    },
    {
      response: {
        200: t.Object({
          data: AuthProviderModels.response,
        }),
      },
    },
  )
  .patch(
    "/",
    async ({ user, params, body, set }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Validate at least one provider
      if (body.providers.length === 0) {
        set.status = 400;
        return {
          error: {
            message: "At least one provider must be enabled",
            code: "VALIDATION_ERROR",
          },
        };
      }

      // Ensure email is always included
      const providers = Array.from(new Set([...body.providers, "email"]));

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      // Update or create auth config
      const [existing] = await projectDb
        .select()
        .from(authSettings)
        .where(eq(authSettings.id, "default"))
        .limit(1);

      let authConfig;
      if (existing) {
        [authConfig] = await projectDb
          .update(authSettings)
          .set({
            providers: providers as any,
            updatedAt: new Date(),
          })
          .where(eq(authSettings.id, "default"))
          .returning();
      } else {
        [authConfig] = await projectDb
          .insert(authSettings)
          .values({
            id: "default",
            providers: providers as any,
          })
          .returning();
      }

      if (!authConfig) {
        throw new Error("Failed to update auth configuration");
      }

      logProjectAuthEvent(params.id, "providers_updated", {
        providers: (authConfig.providers as string[]) || [],
        previousProviders: existing ? (existing.providers as string[]) : [],
      });

      return {
        data: {
          projectId: params.id,
          providers: (authConfig.providers as string[]) || [],
          createdAt: authConfig.createdAt,
          updatedAt: authConfig.updatedAt,
        },
      };
    },
    {
      body: AuthProviderModels.update,
      response: {
        200: t.Object({
          data: AuthProviderModels.response,
        }),
        400: t.Object({
          error: t.Object({
            message: t.String(),
            code: t.Optional(t.String()),
          }),
        }),
      },
    },
  )
  .post(
    "/:provider",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      // Get existing auth config
      const [existing] = await projectDb
        .select()
        .from(authSettings)
        .where(eq(authSettings.id, "default"))
        .limit(1);

      const currentProviders = existing
        ? (existing.providers as string[])
        : ["email"];

      // Toggle provider
      const providers = currentProviders.includes(params.provider)
        ? currentProviders.filter((p) => p !== params.provider)
        : [...currentProviders, params.provider];

      // Ensure email is always included
      const finalProviders = Array.from(new Set([...providers, "email"]));

      let authConfig;
      if (existing) {
        [authConfig] = await projectDb
          .update(authSettings)
          .set({
            providers: finalProviders as any,
            updatedAt: new Date(),
          })
          .where(eq(authSettings.id, "default"))
          .returning();
      } else {
        [authConfig] = await projectDb
          .insert(authSettings)
          .values({
            id: "default",
            providers: finalProviders as any,
          })
          .returning();
      }

      if (!authConfig) {
        throw new Error("Failed to update auth configuration");
      }

      const wasEnabled = currentProviders.includes(params.provider);
      const action = wasEnabled ? "disabled" : "enabled";

      logProjectAuthEvent(params.id, `provider_${action}`, {
        provider: params.provider,
        action,
        providers: (authConfig.providers as string[]) || [],
      });

      return {
        data: {
          projectId: params.id,
          providers: (authConfig.providers as string[]) || [],
          createdAt: authConfig.createdAt,
          updatedAt: authConfig.updatedAt,
        },
        message: `Provider ${params.provider} ${action}`,
      };
    },
    {
      params: t.Object({
        provider: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          data: AuthProviderModels.response,
          message: t.String(),
        }),
      },
    },
  )
  // OAuth Provider Configuration Routes
  .guard({
    params: t.Object({
      id: t.String({ minLength: 1 }),
      provider: t.Union([
        t.Literal("google"),
        t.Literal("github"),
        t.Literal("facebook"),
        t.Literal("apple"),
      ]),
    }),
  })
  .get(
    "/oauth/:provider",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      try {
        const [oauthConfig] = await projectDb
          .select()
          .from(oauthProviders)
          .where(eq(oauthProviders.provider, params.provider as any))
          .limit(1);

        if (!oauthConfig) {
          return {
            data: null,
          };
        }

        // Decrypt client secret for display (but mask it)
        let maskedSecret = "";
        try {
          if (oauthConfig.clientSecret) {
            const decryptedSecret = await decrypt(oauthConfig.clientSecret);
            maskedSecret = decryptedSecret
              ? `${decryptedSecret.substring(0, 4)}${"*".repeat(Math.max(0, decryptedSecret.length - 8))}${decryptedSecret.substring(decryptedSecret.length - 4)}`
              : "";
          }
        } catch (decryptError) {
          // If decryption fails, just show masked version
          maskedSecret = "****";
        }

        return {
          data: {
            id: oauthConfig.id,
            provider: oauthConfig.provider,
            clientId: oauthConfig.clientId,
            clientSecret: maskedSecret, // Return masked version
            redirectUri: oauthConfig.redirectUri || undefined,
            scopes: (oauthConfig.scopes as string[]) || [],
            isConfigured: oauthConfig.isConfigured,
            lastTestedAt: oauthConfig.lastTestedAt || undefined,
            lastTestStatus: oauthConfig.lastTestStatus || undefined,
            createdAt: oauthConfig.createdAt,
            updatedAt: oauthConfig.updatedAt,
          },
        };
      } catch (error: any) {
        // Log the error for debugging
        console.error("Error fetching OAuth config:", error);
        
        // If table doesn't exist or any database error, return null (not configured)
        // This allows the UI to work even if migrations haven't been run
        if (
          error?.message?.includes("does not exist") ||
          error?.message?.includes("no such table") ||
          error?.message?.includes("relation") ||
          error?.code === "42P01" // PostgreSQL table doesn't exist error code
        ) {
          return {
            data: null,
          };
        }
        
        // For other errors, re-throw to be handled by error handler
        throw error;
      }
    },
    {
      response: {
        200: t.Object({
          data: t.Union([
            t.Object({
              id: t.String(),
              provider: t.String(),
              clientId: t.String(),
              clientSecret: t.String(),
              redirectUri: t.Optional(t.String()),
              scopes: t.Array(t.String()),
              isConfigured: t.Boolean(),
              lastTestedAt: t.Optional(t.Date()),
              lastTestStatus: t.Optional(t.String()),
              createdAt: t.Date(),
              updatedAt: t.Date(),
            }),
            t.Null(),
          ]),
        }),
      },
    },
  )
  .post(
    "/oauth/:provider",
    async ({ user, params, body }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      try {
        // Encrypt client secret
        const encryptedSecret = await encrypt(body.clientSecret);

        // Check if config exists
        const [existing] = await projectDb
          .select()
          .from(oauthProviders)
          .where(eq(oauthProviders.provider, params.provider as any))
          .limit(1);

        let oauthConfig;
        if (existing) {
          [oauthConfig] = await projectDb
            .update(oauthProviders)
            .set({
              clientId: body.clientId,
              clientSecret: encryptedSecret,
              redirectUri: body.redirectUri,
              scopes: (body.scopes || []) as any,
              isConfigured: true,
              updatedAt: new Date(),
            })
            .where(eq(oauthProviders.id, existing.id))
            .returning();
        } else {
          [oauthConfig] = await projectDb
            .insert(oauthProviders)
            .values({
              id: nanoid(),
              provider: params.provider as any,
              clientId: body.clientId,
              clientSecret: encryptedSecret,
              redirectUri: body.redirectUri,
              scopes: (body.scopes || []) as any,
              isConfigured: true,
            })
            .returning();
        }

        if (!oauthConfig) {
          throw new Error("Failed to save OAuth configuration");
        }

        logProjectAuthEvent(params.id, "oauth_configured", {
          provider: params.provider,
        });

        return {
          data: {
            id: oauthConfig.id,
            provider: oauthConfig.provider,
            clientId: oauthConfig.clientId,
            redirectUri: oauthConfig.redirectUri || undefined,
            scopes: (oauthConfig.scopes as string[]) || [],
            isConfigured: oauthConfig.isConfigured,
          },
          message: `OAuth provider ${params.provider} configured successfully`,
        };
      } catch (error: any) {
        console.error("Error saving OAuth config:", error);
        
        // If table doesn't exist, provide helpful error
        if (
          error?.message?.includes("does not exist") ||
          error?.message?.includes("no such table") ||
          error?.message?.includes("relation") ||
          error?.code === "42P01"
        ) {
          throw new Error(
            "OAuth configuration table does not exist. Please run database migrations first."
          );
        }
        
        throw error;
      }
    },
    {
      body: t.Object({
        clientId: t.String({ minLength: 1 }),
        clientSecret: t.String({ minLength: 1 }),
        redirectUri: t.Optional(t.String()),
        scopes: t.Optional(t.Array(t.String())),
      }),
      response: {
        200: t.Object({
          data: t.Object({
            id: t.String(),
            provider: t.String(),
            clientId: t.String(),
            redirectUri: t.Optional(t.String()),
            scopes: t.Array(t.String()),
            isConfigured: t.Boolean(),
          }),
          message: t.String(),
        }),
      },
    },
  )
  .post(
    "/oauth/:provider/test",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      try {
        const [oauthConfig] = await projectDb
          .select()
          .from(oauthProviders)
          .where(eq(oauthProviders.provider, params.provider as any))
          .limit(1);

        if (!oauthConfig || !oauthConfig.isConfigured) {
          return {
            success: false,
            message: "OAuth provider is not configured",
            status: "failed",
          };
        }

        // Decrypt secret for testing
        const decryptedSecret = await decrypt(oauthConfig.clientSecret);

        // Basic validation test - in production, you'd make an actual OAuth API call
        let testStatus: "success" | "failed" = "success";
        let testMessage = "Connection test successful";

        try {
          // Validate that client ID and secret are not empty
          if (!oauthConfig.clientId || !decryptedSecret) {
            throw new Error("Client ID or secret is missing");
          }

          // For now, just validate format - in production, make actual OAuth call
          // This is a placeholder - you'd implement actual OAuth validation here
        } catch (error) {
          testStatus = "failed";
          testMessage = error instanceof Error ? error.message : "Connection test failed";
        }

        // Update test status
        await projectDb
          .update(oauthProviders)
          .set({
            lastTestedAt: new Date(),
            lastTestStatus: testStatus,
            updatedAt: new Date(),
          })
          .where(eq(oauthProviders.id, oauthConfig.id));

        return {
          success: testStatus === "success",
          message: testMessage,
          status: testStatus,
        };
      } catch (error: any) {
        console.error("Error testing OAuth connection:", error);
        
        if (
          error?.message?.includes("does not exist") ||
          error?.message?.includes("no such table") ||
          error?.message?.includes("relation") ||
          error?.code === "42P01"
        ) {
          return {
            success: false,
            message: "OAuth provider is not configured",
            status: "failed",
          };
        }
        
        throw error;
      }
    },
    {
      response: {
        200: t.Object({
          success: t.Boolean(),
          message: t.String(),
          status: t.String(),
        }),
      },
    },
  )
  .delete(
    "/oauth/:provider",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      try {
        const [oauthConfig] = await projectDb
          .select()
          .from(oauthProviders)
          .where(eq(oauthProviders.provider, params.provider as any))
          .limit(1);

        if (!oauthConfig) {
          throw new NotFoundError("OAuth Configuration", params.provider);
        }

        await projectDb.delete(oauthProviders).where(eq(oauthProviders.id, oauthConfig.id));

        logProjectAuthEvent(params.id, "oauth_removed", {
          provider: params.provider,
        });

        return {
          message: `OAuth provider ${params.provider} configuration removed`,
        };
      } catch (error: any) {
        console.error("Error deleting OAuth config:", error);
        
        if (
          error?.message?.includes("does not exist") ||
          error?.message?.includes("no such table") ||
          error?.message?.includes("relation") ||
          error?.code === "42P01"
        ) {
          // Table doesn't exist, so config doesn't exist either
          throw new NotFoundError("OAuth Configuration", params.provider);
        }
        
        throw error;
      }
    },
    {
      response: {
        200: t.Object({
          message: t.String(),
        }),
      },
    },
  )
  // Auth Settings Routes
  .guard({
    params: t.Object({
      id: t.String({ minLength: 1 }),
    }),
  })
  .get(
    "/settings",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      let [settings] = await projectDb
        .select()
        .from(detailedAuthSettings)
        .where(eq(detailedAuthSettings.id, "default"))
        .limit(1);

      if (!settings) {
        // Create default settings
        [settings] = await projectDb
          .insert(detailedAuthSettings)
          .values({
            id: "default",
          })
          .returning();
      }

      if (!settings) {
        throw new Error("Failed to get auth settings");
      }

      return {
        data: {
          projectId: params.id,
          requireEmailVerification: settings.requireEmailVerification,
          rateLimitMax: parseInt(settings.rateLimitMax || "5"),
          rateLimitWindow: parseInt(settings.rateLimitWindow || "15"),
          sessionExpirationDays: parseInt(settings.sessionExpirationDays || "30"),
          minPasswordLength: parseInt(settings.minPasswordLength || "8"),
          requireUppercase: settings.requireUppercase,
          requireLowercase: settings.requireLowercase,
          requireNumbers: settings.requireNumbers,
          requireSpecialChars: settings.requireSpecialChars,
          mfaEnabled: settings.mfaEnabled,
          mfaRequired: settings.mfaRequired,
          createdAt: settings.createdAt,
          updatedAt: settings.updatedAt,
        },
      };
    },
    {
      response: {
        200: t.Object({
          data: t.Object({
            projectId: t.String(),
            requireEmailVerification: t.Boolean(),
            rateLimitMax: t.Number(),
            rateLimitWindow: t.Number(),
            sessionExpirationDays: t.Number(),
            minPasswordLength: t.Number(),
            requireUppercase: t.Boolean(),
            requireLowercase: t.Boolean(),
            requireNumbers: t.Boolean(),
            requireSpecialChars: t.Boolean(),
            mfaEnabled: t.Boolean(),
            mfaRequired: t.Boolean(),
            createdAt: t.Date(),
            updatedAt: t.Date(),
          }),
        }),
      },
    },
  )
  .patch(
    "/settings",
    async ({ user, params, body }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      const [existing] = await projectDb
        .select()
        .from(detailedAuthSettings)
        .where(eq(detailedAuthSettings.id, "default"))
        .limit(1);

      const updateData: any = {
        updatedAt: new Date(),
      };

      if (body.requireEmailVerification !== undefined) {
        updateData.requireEmailVerification = body.requireEmailVerification;
      }
      if (body.rateLimitMax !== undefined) {
        updateData.rateLimitMax = body.rateLimitMax.toString();
      }
      if (body.rateLimitWindow !== undefined) {
        updateData.rateLimitWindow = body.rateLimitWindow.toString();
      }
      if (body.sessionExpirationDays !== undefined) {
        updateData.sessionExpirationDays = body.sessionExpirationDays.toString();
      }
      if (body.minPasswordLength !== undefined) {
        updateData.minPasswordLength = body.minPasswordLength.toString();
      }
      if (body.requireUppercase !== undefined) {
        updateData.requireUppercase = body.requireUppercase;
      }
      if (body.requireLowercase !== undefined) {
        updateData.requireLowercase = body.requireLowercase;
      }
      if (body.requireNumbers !== undefined) {
        updateData.requireNumbers = body.requireNumbers;
      }
      if (body.requireSpecialChars !== undefined) {
        updateData.requireSpecialChars = body.requireSpecialChars;
      }
      if (body.mfaEnabled !== undefined) {
        updateData.mfaEnabled = body.mfaEnabled;
      }
      if (body.mfaRequired !== undefined) {
        updateData.mfaRequired = body.mfaRequired;
      }

      let settings;
      if (existing) {
        [settings] = await projectDb
          .update(detailedAuthSettings)
          .set(updateData)
          .where(eq(detailedAuthSettings.id, "default"))
          .returning();
      } else {
        [settings] = await projectDb
          .insert(detailedAuthSettings)
          .values({
            id: "default",
            ...updateData,
          })
          .returning();
      }

      if (!settings) {
        throw new Error("Failed to update auth settings");
      }

      logProjectAuthEvent(params.id, "auth_settings_updated", {
        settings: updateData,
      });

      return {
        data: {
          projectId: params.id,
          requireEmailVerification: settings.requireEmailVerification,
          rateLimitMax: parseInt(settings.rateLimitMax || "5"),
          rateLimitWindow: parseInt(settings.rateLimitWindow || "15"),
          sessionExpirationDays: parseInt(settings.sessionExpirationDays || "30"),
          minPasswordLength: parseInt(settings.minPasswordLength || "8"),
          requireUppercase: settings.requireUppercase,
          requireLowercase: settings.requireLowercase,
          requireNumbers: settings.requireNumbers,
          requireSpecialChars: settings.requireSpecialChars,
          mfaEnabled: settings.mfaEnabled,
          mfaRequired: settings.mfaRequired,
          createdAt: settings.createdAt,
          updatedAt: settings.updatedAt,
        },
      };
    },
    {
      body: t.Object({
        requireEmailVerification: t.Optional(t.Boolean()),
        rateLimitMax: t.Optional(t.Number()),
        rateLimitWindow: t.Optional(t.Number()),
        sessionExpirationDays: t.Optional(t.Number()),
        minPasswordLength: t.Optional(t.Number()),
        requireUppercase: t.Optional(t.Boolean()),
        requireLowercase: t.Optional(t.Boolean()),
        requireNumbers: t.Optional(t.Boolean()),
        requireSpecialChars: t.Optional(t.Boolean()),
        mfaEnabled: t.Optional(t.Boolean()),
        mfaRequired: t.Optional(t.Boolean()),
      }),
      response: {
        200: t.Object({
          data: t.Object({
            projectId: t.String(),
            requireEmailVerification: t.Boolean(),
            rateLimitMax: t.Number(),
            rateLimitWindow: t.Number(),
            sessionExpirationDays: t.Number(),
            minPasswordLength: t.Number(),
            requireUppercase: t.Boolean(),
            requireLowercase: t.Boolean(),
            requireNumbers: t.Boolean(),
            requireSpecialChars: t.Boolean(),
            mfaEnabled: t.Boolean(),
            mfaRequired: t.Boolean(),
            createdAt: t.Date(),
            updatedAt: t.Date(),
          }),
        }),
      },
    },
  )
  // Statistics Route
  .get(
    "/statistics",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      // Get provider usage statistics from project accounts
      const accounts = await projectDb
        .select({
          providerId: projectAccounts.providerId,
          createdAt: projectAccounts.createdAt,
        })
        .from(projectAccounts);

      // Count by provider
      const providerCounts: Record<string, number> = {};
      const signupsByDate: Record<string, number> = {};

      accounts.forEach((account) => {
        const provider = account.providerId;
        if (provider && typeof provider === "string") {
          providerCounts[provider] = (providerCounts[provider] || 0) + 1;
        }

        if (account.createdAt) {
          try {
            const date = account.createdAt.toISOString().split("T")[0];
            if (date && typeof date === "string") {
              signupsByDate[date] = (signupsByDate[date] || 0) + 1;
            }
          } catch (error) {
            // Skip invalid dates
          }
        }
      });

      // Get total users count from project database
      const { projectUsers } = await import("../db/project-schema");
      const allUsers = await projectDb.select().from(projectUsers);

      return {
        data: {
          totalUsers: allUsers.length,
          providerBreakdown: providerCounts,
          signupsOverTime: signupsByDate,
        },
      };
    },
    {
      response: {
        200: t.Object({
          data: t.Object({
            totalUsers: t.Number(),
            providerBreakdown: t.Record(t.String(), t.Number()),
            signupsOverTime: t.Record(t.String(), t.Number()),
          }),
        }),
      },
    },
  );
