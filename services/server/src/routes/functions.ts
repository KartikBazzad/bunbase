import { Elysia, t } from "elysia";
import { apiKeyResolver } from "../middleware/api-key";
import { NotFoundError } from "../lib/errors";
import {
  getProjectFunctions,
  getFunctionById,
  createFunction,
  updateFunction,
  deleteFunction,
  getFunctionVersions,
} from "../lib/function-helpers";
import { executeFunction } from "../lib/function-executor";
import { FunctionModels, CommonModels } from "./models";
import {
  db,
  functionEnvironments,
  functionLogs,
  functionMetrics,
} from "../db";
import { functions } from "../db/schema";
import { eq, and, desc, sql } from "drizzle-orm";
import { encrypt, decrypt } from "../lib/encryption";
import { nanoid } from "nanoid";
import { readFunctionCode } from "../lib/function-storage";
import { validateFunctionCode } from "../lib/function-validator";
import { logProjectOperation } from "../lib/project-logger-utils";
import { getFunctionLogs as getFunctionLogsFromStorage } from "../lib/function-log-storage";
import { functionVersions } from "../db/schema";

export const functionsRoutes = new Elysia({ prefix: "/functions" })
  .resolve(apiKeyResolver)
  .model({
    "function.create": FunctionModels.create,
    "function.update": FunctionModels.update,
    "function.response": FunctionModels.response,
    "function.listResponse": FunctionModels.listResponse,
    "function.invoke": FunctionModels.invoke,
    "function.env": FunctionModels.env,
    "function.deployResponse": FunctionModels.deployResponse,
    "function.metricsResponse": FunctionModels.metricsResponse,
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
    if (error instanceof Error) {
      set.status = 500;
      return {
        error: {
          message: error.message,
          code: "INTERNAL_ERROR",
        },
      };
    }
  })
  // List all functions
  .get(
    "/",
    async ({ apiKey }) => {
      const projectFunctions = await getProjectFunctions(apiKey.projectId);

      return projectFunctions.map((fn) => ({
        id: fn.id,
        name: fn.name,
        runtime: fn.runtime,
        handler: fn.handler,
        status: fn.status,
        memory: fn.memory,
        timeout: fn.timeout,
        createdAt: fn.createdAt,
        updatedAt: fn.updatedAt,
      }));
    },
    {
      response: {
        200: FunctionModels.listResponse,
      },
    },
  )
  // Create function
  .post(
    "/",
    async ({ apiKey, body }) => {
      const func = await createFunction(apiKey.projectId, {
        name: body.name,
        runtime: body.runtime || "bun",
        handler: body.handler,
        code: body.code,
        memory: body.memory,
        timeout: body.timeout,
      });

      return {
        id: func.id,
        name: func.name,
        runtime: func.runtime,
        handler: func.handler,
        status: func.status,
        memory: func.memory,
        timeout: func.timeout,
        createdAt: func.createdAt,
        updatedAt: func.updatedAt,
      };
    },
    {
      body: FunctionModels.create,
      response: {
        200: FunctionModels.response,
      },
    },
  )
  // Get function details
  .get(
    "/:id",
    async ({ apiKey, params }) => {
      const func = await getFunctionById(params.id, apiKey.projectId);

      return {
        id: func.id,
        name: func.name,
        runtime: func.runtime,
        handler: func.handler,
        status: func.status,
        memory: func.memory,
        timeout: func.timeout,
        createdAt: func.createdAt,
        updatedAt: func.updatedAt,
      };
    },
    {
      params: FunctionModels.params,
      response: {
        200: FunctionModels.response,
      },
    },
  )
  // Update function
  .put(
    "/:id",
    async ({ apiKey, params, body }) => {
      const updated = await updateFunction(params.id, apiKey.projectId, {
        name: body.name,
        runtime: body.runtime,
        handler: body.handler,
        code: body.code,
        memory: body.memory,
        timeout: body.timeout,
      });

      return {
        id: updated.id,
        name: updated.name,
        runtime: updated.runtime,
        handler: updated.handler,
        status: updated.status,
        memory: updated.memory,
        timeout: updated.timeout,
        createdAt: updated.createdAt,
        updatedAt: updated.updatedAt,
      };
    },
    {
      params: FunctionModels.params,
      body: FunctionModels.update,
      response: {
        200: FunctionModels.response,
      },
    },
  )
  // Delete function
  .delete(
    "/:id",
    async ({ apiKey, params }) => {
      await deleteFunction(params.id, apiKey.projectId);

      return {
        message: "Function deleted successfully",
      };
    },
    {
      params: FunctionModels.params,
      response: {
        200: CommonModels.success,
      },
    },
  )
  // Deploy function
  .post(
    "/:id/deploy",
    async ({ apiKey, params }) => {
      const func = await getFunctionById(params.id, apiKey.projectId);

      // Get latest version
      const versions = await getFunctionVersions(params.id, apiKey.projectId);
      if (versions.length === 0) {
        throw new Error("No code version available to deploy");
      }

      const latestVersion = versions[0];
      
      // If function already has an active version, verify it exists
      if (func.activeVersionId) {
        const [activeVersion] = await db
          .select()
          .from(functionVersions)
          .where(eq(functionVersions.id, func.activeVersionId))
          .limit(1);
        
        if (!activeVersion) {
          // Active version was deleted, use latest
        } else if (activeVersion.id !== latestVersion.id) {
          // Different version, will update below
        }
      }

      // Validate code before deployment
      const code = await readFunctionCode(
        apiKey.projectId,
        params.id,
        latestVersion.version,
      );
      const validation = await validateFunctionCode(code, func.runtime);
      if (!validation.valid) {
        throw new Error(
          `Code validation failed: ${validation.errors.join(", ")}`,
        );
      }

      // Deactivate existing deployments
      await db
        .update(functionDeployments)
        .set({ status: "inactive" })
        .where(eq(functionDeployments.functionId, params.id));

      // Create new deployment
      const deploymentId = nanoid();
      await db.insert(functionDeployments).values({
        id: deploymentId,
        functionId: params.id,
        versionId: latestVersion.id,
        environment: "production",
        status: "active",
      });

      // Update function status and set active version
      const { functions: functionsTable } = await import("../db/schema");
      await db
        .update(functionsTable)
        .set({
          status: "deployed",
          activeVersionId: latestVersion.id,
          updatedAt: new Date(),
        })
        .where(eq(functionsTable.id, params.id));

      // activeVersionId is already set in the update above

      // Log deployment
      logProjectOperation(apiKey.projectId, "function_deploy", {
        functionId: params.id,
        functionName: func.name,
        version: latestVersion.version,
        deploymentId,
      });

      return {
        message: "Function deployed successfully",
        version: latestVersion.version,
        deploymentId,
      };
    },
    {
      params: FunctionModels.params,
      response: {
        200: FunctionModels.deployResponse,
      },
    },
  )
  // Rollback deployment
  .post(
    "/:id/rollback",
    async ({ apiKey, params }) => {
      const func = await getFunctionById(params.id, apiKey.projectId);

      // Get all versions
      const versions = await getFunctionVersions(params.id, apiKey.projectId);
      if (versions.length < 2) {
        throw new Error("No previous version to rollback to");
      }

      // Get previous version (skip current)
      const previousVersion = versions[1];

      // Deactivate existing deployments
      await db
        .update(functionDeployments)
        .set({ status: "inactive" })
        .where(eq(functionDeployments.functionId, params.id));

      // Create new deployment with previous version
      const deploymentId = nanoid();
      await db.insert(functionDeployments).values({
        id: deploymentId,
        functionId: params.id,
        versionId: previousVersion.id,
        environment: "production",
        status: "active",
      });

      // Update active version
      const { functions: functionsTable } = await import("../db/schema");
      await db
        .update(functionsTable)
        .set({
          activeVersionId: previousVersion.id,
          updatedAt: new Date(),
        })
        .where(eq(functionsTable.id, params.id));

      return {
        message: "Function rolled back successfully",
        version: previousVersion.version,
        deploymentId,
      };
    },
    {
      params: FunctionModels.params,
      response: {
        200: FunctionModels.deployResponse,
      },
    },
  )
  // Invoke function
  .post(
    "/:id/invoke",
    async ({ apiKey, params, body, request }) => {
      const func = await getFunctionById(params.id, apiKey.projectId);

      if (func.status !== "deployed") {
        throw new Error("Function is not deployed");
      }

      // Execute function
      const result = await executeFunction(
        params.id,
        apiKey.projectId,
        {
          method: body.method || request.method || "POST",
          url: body.url || request.url || "/",
          headers: body.headers || {},
          body: body.body,
        },
      );

      if (!result.success) {
        throw new Error(result.error || "Function execution failed");
      }

      return {
        result: result.response?.body,
        executionTime: result.executionTime,
        executionId: result.executionId,
      };
    },
    {
      params: FunctionModels.params,
      body: FunctionModels.invoke,
      response: {
        200: t.Object({
          result: t.Any(),
          executionTime: t.Number(),
        }),
      },
    },
  )
  // Get function logs
  .get(
    "/:id/logs",
    async ({ apiKey, params, query }) => {
      await getFunctionById(params.id, apiKey.projectId);

      const limit = query.limit ? parseInt(query.limit as string) : 100;
      const offset = query.offset ? parseInt(query.offset as string) : 0;
      const level = query.level as string | undefined;
      const executionId = query.executionId as string | undefined;
      const startDate = query.startDate
        ? new Date(query.startDate as string)
        : undefined;
      const endDate = query.endDate
        ? new Date(query.endDate as string)
        : undefined;

      // Get logs from SQLite storage
      const logs = await getFunctionLogsFromStorage(apiKey.projectId, params.id, {
        limit,
        offset,
        level,
        executionId,
        startDate,
        endDate,
      });

      return {
        logs: logs.map((log) => ({
          id: log.id,
          executionId: log.executionId,
          level: log.level,
          message: log.message,
          metadata: log.metadata,
          timestamp: log.timestamp,
        })),
        total: logs.length, // Note: total count would require separate query
      };
    },
    {
      params: FunctionModels.params,
      query: t.Object({
        limit: t.Optional(t.String()),
        offset: t.Optional(t.String()),
        level: t.Optional(t.String()),
        executionId: t.Optional(t.String()),
        startDate: t.Optional(t.String()),
        endDate: t.Optional(t.String()),
      }),
      response: {
        200: t.Object({
          logs: t.Array(t.Any()),
          total: t.Number(),
        }),
      },
    },
  )
  // Get function metrics
  .get(
    "/:id/metrics",
    async ({ apiKey, params, query }) => {
      await getFunctionById(params.id, apiKey.projectId);

      const period = (query.period as string) || "day"; // minute, hour, day

      if (period === "minute") {
        // Get minute-level metrics for last hour
        const oneHourAgo = new Date();
        oneHourAgo.setHours(oneHourAgo.getHours() - 1);

        const minuteMetrics = await db
          .select()
          .from(functionMetricsMinute)
          .where(
            and(
              eq(functionMetricsMinute.functionId, params.id),
              // Note: Need to add timestamp comparison - for now get all recent
            ),
          )
          .orderBy(desc(functionMetricsMinute.timestamp))
          .limit(60); // Last 60 minutes

        const totalInvocations = minuteMetrics.reduce(
          (sum, m) => sum + m.invocations,
          0,
        );
        const totalErrors = minuteMetrics.reduce(
          (sum, m) => sum + m.errors,
          0,
        );
        const totalDuration = minuteMetrics.reduce(
          (sum, m) => sum + m.totalDuration,
          0,
        );

        return {
          invocations: totalInvocations,
          errors: totalErrors,
          averageDuration:
            totalInvocations > 0
              ? Math.round(totalDuration / totalInvocations)
              : 0,
          lastInvoked: minuteMetrics[0]?.timestamp || null,
          period: "minute",
          dataPoints: minuteMetrics.length,
        };
      }

      // Default: daily metrics
      const today = new Date();
      today.setHours(0, 0, 0, 0);

      const [metric] = await db
        .select()
        .from(functionMetrics)
        .where(
          and(
            eq(functionMetrics.functionId, params.id),
            eq(functionMetrics.date, today),
          ),
        )
        .limit(1);

      // Get last invocation from minute metrics
      const [lastMinuteMetric] = await db
        .select()
        .from(functionMetricsMinute)
        .where(eq(functionMetricsMinute.functionId, params.id))
        .orderBy(desc(functionMetricsMinute.timestamp))
        .limit(1);

      const invocations = metric?.invocations || 0;
      const errors = metric?.errors || 0;
      const totalDuration = metric?.totalDuration || 0;
      const averageDuration =
        invocations > 0 ? totalDuration / invocations : 0;

      return {
        invocations,
        errors,
        averageDuration: Math.round(averageDuration),
        lastInvoked: lastMinuteMetric?.timestamp || null,
        period: "day",
      };
    },
    {
      params: FunctionModels.params,
      query: t.Object({
        period: t.Optional(t.String()), // minute, hour, day
      }),
      response: {
        200: FunctionModels.metricsResponse,
      },
    },
  )
  // List env variables
  .get(
    "/:id/env",
    async ({ apiKey, params }) => {
      await getFunctionById(params.id, apiKey.projectId);

      const envVars = await db
        .select()
        .from(functionEnvironments)
        .where(eq(functionEnvironments.functionId, params.id));

      // Decrypt secret values
      const decrypted = await Promise.all(
        envVars.map(async (env) => {
          const value = env.isSecret ? await decrypt(env.value) : env.value;
          return {
            key: env.key,
            value,
            isSecret: env.isSecret,
          };
        }),
      );

      return decrypted;
    },
    {
      params: FunctionModels.params,
      response: {
        200: t.Array(
          t.Object({
            key: t.String(),
            value: t.String(),
            isSecret: t.Optional(t.Boolean()),
          }),
        ),
      },
    },
  )
  // Set env variable
  .post(
    "/:id/env",
    async ({ apiKey, params, body }) => {
      await getFunctionById(params.id, apiKey.projectId);

      // Check if env var already exists
      const [existing] = await db
        .select()
        .from(functionEnvironments)
        .where(
          and(
            eq(functionEnvironments.functionId, params.id),
            eq(functionEnvironments.key, body.key),
          ),
        )
        .limit(1);

      const encryptedValue = await encrypt(body.value);
      const isSecret = body.key.toLowerCase().includes("secret") ||
        body.key.toLowerCase().includes("key") ||
        body.key.toLowerCase().includes("password") ||
        body.key.toLowerCase().includes("token");

      if (existing) {
        // Update existing
        await db
          .update(functionEnvironments)
          .set({
            value: encryptedValue,
            isSecret,
          })
          .where(eq(functionEnvironments.id, existing.id));
      } else {
        // Create new
        await db.insert(functionEnvironments).values({
          id: nanoid(),
          functionId: params.id,
          key: body.key,
          value: encryptedValue,
          isSecret,
        });
      }

      return {
        message: "Environment variable set successfully",
      };
    },
    {
      params: FunctionModels.params,
      body: FunctionModels.env,
      response: {
        200: CommonModels.success,
      },
    },
  )
  // Delete env variable
  .delete(
    "/:id/env/:key",
    async ({ apiKey, params }) => {
      await getFunctionById(params.id, apiKey.projectId);

      await db
        .delete(functionEnvironments)
        .where(
          and(
            eq(functionEnvironments.functionId, params.id),
            eq(functionEnvironments.key, params.key),
          ),
        );

      return {
        message: "Environment variable deleted successfully",
      };
    },
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
        key: t.String({ minLength: 1 }),
      }),
      response: {
        200: CommonModels.success,
      },
    },
  );
