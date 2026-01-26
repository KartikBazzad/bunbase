import { Elysia, t } from "elysia";
import { db } from "../db";
import { authResolver } from "../middleware/auth";
import { NotFoundError, ForbiddenError } from "../lib/errors";
import { requireAuth } from "../lib/auth-helpers";
import { eq, or, like, desc, and, inArray } from "drizzle-orm";
import { nanoid } from "nanoid";
import { auth } from "../auth";
import { logger } from "../lib/logger";
import { getProjectDb } from "../db/project-db-helpers";
import { projectUsers, projectAccounts } from "../db/project-schema";

// Helper function to verify project ownership
async function verifyProjectOwnership(
  projectId: string,
  userId: string,
): Promise<any> {
  const { projects } = await import("../db");
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

export const usersRoutes = new Elysia({
  prefix: "/projects/:id/users",
})
  .resolve(authResolver)
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
    async ({ user, params, query }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      const limit = query.limit ? parseInt(query.limit) : 20;
      const offset = query.offset ? parseInt(query.offset) : 0;
      const search = query.search || "";
      const isEmailVerified = query.emailVerified ? query.emailVerified === "true" : undefined;

      const conditions = [];

      if (search) {
        conditions.push(
          or(
            like(projectUsers.name, `%${search}%`),
            like(projectUsers.email, `%${search}%`)
          )
        );
      }

      if (isEmailVerified !== undefined) {
        conditions.push(eq(projectUsers.emailVerified, isEmailVerified));
      }

      const userList = await projectDb
        .select()
        .from(projectUsers)
        .where(conditions.length > 0 ? and(...conditions) : undefined)
        .orderBy(desc(projectUsers.createdAt))
        .limit(limit)
        .offset(offset);

      // Get total count
      const allUsers = await projectDb
        .select()
        .from(projectUsers)
        .where(conditions.length > 0 ? and(...conditions) : undefined);

      // Get provider info from accounts for all users
      const userIds = userList.map((u) => u.id);
      const accounts = userIds.length > 0
        ? await projectDb
            .select()
            .from(projectAccounts)
            .where(inArray(projectAccounts.userId, userIds))
        : [];

      // Create a map of userId -> provider (use first account for each user)
      const providerMap = new Map<string, string>();
      for (const account of accounts) {
        if (!providerMap.has(account.userId)) {
          providerMap.set(account.userId, account.providerId);
        }
      }

      return {
        data: userList.map((u) => ({
          id: u.id,
          name: u.name,
          email: u.email,
          emailVerified: u.emailVerified,
          provider: providerMap.get(u.id) || "email",
          image: u.image,
          createdAt: u.createdAt,
          updatedAt: u.updatedAt,
        })),
        total: allUsers.length,
        limit,
        offset,
      };
    },
    {
      query: t.Object({
        limit: t.Optional(t.String()),
        offset: t.Optional(t.String()),
        search: t.Optional(t.String()),
        emailVerified: t.Optional(t.String()),
      }),
    },
  )
  .get(
    "/:userId",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      const [userRecord] = await projectDb
        .select()
        .from(projectUsers)
        .where(eq(projectUsers.id, params.userId))
        .limit(1);

      if (!userRecord) {
        throw new NotFoundError("User", params.userId);
      }

      // Get provider from accounts
      const [account] = await projectDb
        .select()
        .from(projectAccounts)
        .where(eq(projectAccounts.userId, params.userId))
        .limit(1);

      return {
        data: {
          id: userRecord.id,
          name: userRecord.name,
          email: userRecord.email,
          emailVerified: userRecord.emailVerified,
          provider: account?.providerId || "email",
          image: userRecord.image,
          createdAt: userRecord.createdAt,
          updatedAt: userRecord.updatedAt,
        },
      };
    },
  )
  .post(
    "/",
    async ({ user, params, body }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      const existingUser = await projectDb
        .select()
        .from(projectUsers)
        .where(eq(projectUsers.email, body.email))
        .limit(1);

      if (existingUser.length > 0) {
        throw new Error("User with this email already exists");
      }

      const userId = nanoid();
      const hashedPassword = await auth.api.hashPassword({ password: body.password });

      await projectDb.transaction(async (tx) => {
        const [newUser] = await tx
          .insert(projectUsers)
          .values({
            id: userId,
            name: body.name,
            email: body.email,
            emailVerified: false,
          })
          .returning();

        await tx.insert(projectAccounts).values({
          id: nanoid(),
          userId: userId,
          accountId: userId,
          providerId: "email",
          password: hashedPassword,
        });

        if (body.sendWelcomeEmail) {
          const { sendWelcomeEmail } = await import("../lib/email-service");
          await sendWelcomeEmail(body.email, body.name);
        }
      });

      const [createdUser] = await projectDb
        .select()
        .from(projectUsers)
        .where(eq(projectUsers.id, userId))
        .limit(1);

      logger.info("User created", { userId, email: body.email, projectId: params.id });

      return {
        data: {
          id: createdUser!.id,
          name: createdUser!.name,
          email: createdUser!.email,
          emailVerified: createdUser!.emailVerified,
          image: createdUser!.image,
          createdAt: createdUser!.createdAt,
          updatedAt: createdUser!.updatedAt,
        },
      };
    },
    {
      body: t.Object({
        name: t.String({ minLength: 1, maxLength: 255 }),
        email: t.String({ format: "email" }),
        password: t.String({ minLength: 8 }),
        sendWelcomeEmail: t.Optional(t.Boolean()),
      }),
    },
  )
  .patch(
    "/:userId",
    async ({ user, params, body }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      const [existingUser] = await projectDb
        .select()
        .from(projectUsers)
        .where(eq(projectUsers.id, params.userId))
        .limit(1);

      if (!existingUser) {
        throw new NotFoundError("User", params.userId);
      }

      const updateData: any = {};

      if (body.name !== undefined) {
        updateData.name = body.name;
      }

      if (body.email !== undefined) {
        const emailCheck = await projectDb
          .select()
          .from(projectUsers)
          .where(eq(projectUsers.email, body.email))
          .limit(1);

        if (emailCheck.length > 0 && emailCheck[0].id !== params.userId) {
          throw new Error("User with this email already exists");
        }
        updateData.email = body.email;
      }

      if (Object.keys(updateData).length === 0) {
        throw new Error("No fields to update");
      }

      const [updatedUser] = await projectDb
        .update(projectUsers)
        .set({
          ...updateData,
          updatedAt: new Date(),
        })
        .where(eq(projectUsers.id, params.userId))
        .returning();

      logger.info("User updated", { userId: params.userId, projectId: params.id });

      return {
        data: {
          id: updatedUser.id,
          name: updatedUser.name,
          email: updatedUser.email,
          emailVerified: updatedUser.emailVerified,
          image: updatedUser.image,
          createdAt: updatedUser.createdAt,
          updatedAt: updatedUser.updatedAt,
        },
      };
    },
    {
      body: t.Object({
        name: t.Optional(t.String({ minLength: 1, maxLength: 255 })),
        email: t.Optional(t.String({ format: "email" })),
      }),
    },
  )
  .delete(
    "/:userId",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      const [existingUser] = await projectDb
        .select()
        .from(projectUsers)
        .where(eq(projectUsers.id, params.userId))
        .limit(1);

      if (!existingUser) {
        throw new NotFoundError("User", params.userId);
      }

      // Delete user accounts (cascade will handle user deletion)
      await projectDb.delete(projectAccounts).where(eq(projectAccounts.userId, params.userId));
      // Delete user (accounts are already deleted above, but this ensures user is deleted)
      await projectDb.delete(projectUsers).where(eq(projectUsers.id, params.userId));

      logger.info("User deleted", { userId: params.userId, projectId: params.id });

      return {
        message: "User deleted successfully",
      };
    },
  )
  .post(
    "/:userId/resend-verification",
    async ({ user, params }) => {
      requireAuth(user);
      await verifyProjectOwnership(params.id, user.id);

      // Get project-specific database
      const projectDb = await getProjectDb(params.id);

      const [existingUser] = await projectDb
        .select()
        .from(projectUsers)
        .where(eq(projectUsers.id, params.userId))
        .limit(1);

      if (!existingUser) {
        throw new NotFoundError("User", params.userId);
      }

      if (existingUser.emailVerified) {
        throw new Error("User is already verified");
      }

      // Note: Verifications table might be in the main db for Better Auth
      // For project-specific verification, you might need to create a project verifications table
      // For now, we'll use the main db for verifications
      const { verifications } = await import("../db");

      const verificationToken = nanoid(32);
      const verificationExpiresAt = new Date(Date.now() + 24 * 60 * 60 * 1000); // 24 hours

      await db.insert(verifications).values({
        id: nanoid(),
        identifier: existingUser.email,
        value: verificationToken,
        expiresAt: verificationExpiresAt,
      });

      const { sendVerificationEmail } = await import("../lib/email-service");
      const verificationUrl = `${process.env.BETTER_AUTH_URL || "http://localhost:3000"}/auth/verify-email?token=${verificationToken}`;

      await sendVerificationEmail({
        email: existingUser.email,
        verificationUrl,
        token: verificationToken,
      });

      logger.info("Verification email resent", { userId: params.userId, projectId: params.id });

      return {
        message: "Verification email sent successfully",
      };
    },
  );
