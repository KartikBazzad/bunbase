import { Elysia, t, type Context } from "elysia";
import { authResolver, type AuthenticatedUser } from "../middleware/auth";
import { requireAuth } from "../lib/auth-helpers";
import { UnauthorizedError, NotFoundError } from "../lib/errors";
import { logger } from "../lib/logger";
import { auth } from "../auth";
import {
  bunstoreEvents,
  type DocumentEventPayload,
} from "../lib/bunstore-events";
import { apiKeyResolver, type ApiKeyContext } from "../middleware/api-key";
import type { ServerWebSocket } from "bun";

// WebSocket data type
interface WebSocketData {
  query?: {
    projectId?: string;
  };
  headers?: Record<string, string | undefined>;
}

// Connection management
interface Connection {
  id: string;
  userId?: string; // Optional for API key auth
  apiKey?: ApiKeyContext; // API key context if authenticated via API key
  projectId?: string;
  channels: Set<string>;
  connectedAt: Date;
  lastPing: Date;
  ws?: ServerWebSocket<WebSocketData>;
  authType: "session" | "apiKey"; // Track authentication type
}

const connections = new Map<string, Connection>();
const channels = new Map<string, Set<string>>(); // channel -> connection IDs
const wsInstances = new Map<string, ServerWebSocket<WebSocketData>>(); // connection ID -> WebSocket instance
const pendingApiKeyAuth = new Map<string, ApiKeyContext>(); // Temporary storage for API key auth
const pendingSessionAuth = new Map<string, AuthenticatedUser>(); // Temporary storage for session auth

// Helper to convert Headers to Record
function headersToRecord(headers: Headers): Record<string, string | undefined> {
  const record: Record<string, string | undefined> = {};
  headers.forEach((value, key) => {
    record[key] = value;
  });
  return record;
}

// Heartbeat interval (30 seconds)
const HEARTBEAT_INTERVAL = 30000;
const HEARTBEAT_TIMEOUT = 60000; // 60 seconds timeout

// Start heartbeat checker
setInterval(() => {
  const now = Date.now();
  for (const [connectionId, connection] of connections.entries()) {
    const timeSinceLastPing = now - connection.lastPing.getTime();
    if (timeSinceLastPing > HEARTBEAT_TIMEOUT) {
      // Connection is dead, remove it
      connections.delete(connectionId);
      wsInstances.delete(connectionId);
      // Remove from all channels
      for (const channel of connection.channels) {
        const channelConnections = channels.get(channel);
        if (channelConnections) {
          channelConnections.delete(connectionId);
          if (channelConnections.size === 0) {
            channels.delete(channel);
          }
        }
      }
    }
  }
}, HEARTBEAT_INTERVAL);

// Listen to BunStore document events and broadcast to WebSocket clients
bunstoreEvents.onCreated((payload: DocumentEventPayload) => {
  broadcastDocumentEvent("INSERT", payload);
});

bunstoreEvents.onUpdated((payload: DocumentEventPayload) => {
  broadcastDocumentEvent("UPDATE", payload);
});

bunstoreEvents.onDeleted((payload: DocumentEventPayload) => {
  broadcastDocumentEvent("DELETE", payload);
});

/**
 * Broadcast document event to subscribed WebSocket clients
 */
function broadcastDocumentEvent(
  type: "INSERT" | "UPDATE" | "DELETE",
  payload: DocumentEventPayload,
): void {
  const collectionChannel = `db:${payload.collectionPath}`;
  const documentChannel = `db:${payload.path}`;

  const message = {
    type,
    channel: collectionChannel,
    message: {
      documentId: payload.documentId,
      path: payload.path,
      collectionPath: payload.collectionPath,
      data: payload.data,
      oldData: payload.oldData,
      createdAt: payload.createdAt?.toISOString(),
      updatedAt: payload.updatedAt?.toISOString(),
    },
    timestamp: Date.now(),
  };

  // Broadcast to collection channel subscribers
  const collectionSubscribers = channels.get(collectionChannel);
  if (collectionSubscribers) {
    for (const connId of collectionSubscribers) {
      const ws = wsInstances.get(connId);
      if (ws && ws.readyState === 1) {
        // WebSocket.OPEN = 1
        try {
          ws.send(JSON.stringify(message));
        } catch (error) {
          logger.error("Failed to send message to WebSocket client", error, {
            connectionId: connId,
          });
        }
      }
    }
  }

  // Broadcast to document-specific channel subscribers
  const documentSubscribers = channels.get(documentChannel);
  if (documentSubscribers) {
    for (const connId of documentSubscribers) {
      const ws = wsInstances.get(connId);
      if (ws && ws.readyState === 1) {
        // WebSocket.OPEN = 1
        try {
          ws.send(JSON.stringify({ ...message, channel: documentChannel }));
        } catch (error) {
          logger.error("Failed to send message to WebSocket client", error, {
            connectionId: connId,
          });
        }
      }
    }
  }
}

export const realtimeRoutes = new Elysia({ prefix: "/realtime" })
  .ws("/", {
    // Authenticate connection using either session or API key
    beforeHandle: async ({ status, request }) => {
      const headers = request.headers;
      try {
        // First, try session-based auth (for dashboard)
        try {
          const session = await auth.api.getSession({ headers });
          if (session?.user) {
            // Session auth successful, store for open handler
            const connectionId =
              headers.get("sec-websocket-key") ||
              headers.get("x-connection-id") ||
              `conn-${Date.now()}-${Math.random()}`;
            const authenticatedUser: AuthenticatedUser = {
              id: session.user.id,
              email: session.user.email,
              name: session.user.name,
              emailVerified: session.user.emailVerified,
              image: session.user.image,
            };
            pendingSessionAuth.set(connectionId, authenticatedUser);
            return;
          }
        } catch {
          // Session auth failed, try API key
        }

        // Try API key authentication
        // Create a minimal context-like object for apiKeyResolver
        const headersRecord = headersToRecord(headers);
        const apiKeyContext: Partial<Context> = {
          request,
          headers: headersRecord,
          status,
        };
        const apiKeyResult = await apiKeyResolver(apiKeyContext as Context);

        if (
          apiKeyResult &&
          typeof apiKeyResult === "object" &&
          "apiKey" in apiKeyResult
        ) {
          // API key auth successful
          // Store in temporary map (will be retrieved in open handler)
          // Use a unique identifier from the request
          const connectionId =
            headers.get("sec-websocket-key") ||
            headers.get("x-connection-id") ||
            `conn-${Date.now()}-${Math.random()}`;
          // Store API key context with proper structure
          const apiKeyContext: ApiKeyContext = {
            apiKeyId: apiKeyResult.apiKey.id,
            applicationId: apiKeyResult.apiKey.applicationId,
            projectId: apiKeyResult.apiKey.projectId,
          };
          pendingApiKeyAuth.set(connectionId, apiKeyContext);
          return;
        }

        // Both auth methods failed
        return status(401, {
          error: {
            message:
              "Unauthorized: No valid session or API key found. Provide API key in X-API-Key header or Authorization: Bearer <key>",
            code: "UNAUTHORIZED",
          },
        });
      } catch (error) {
        logger.error("WebSocket authentication error", error);
        return status(401, {
          error: {
            message: "Unauthorized: Authentication failed",
            code: "UNAUTHORIZED",
          },
        });
      }
    },
    open(ws) {
      const connectionId = ws.id;
      
      // Extract projectId from query params if provided
      const projectId = ws.data.query?.projectId;

      // Check authentication from beforeHandle
      // Get sec-websocket-key from headers to retrieve pending auth
      const secWebSocketKey = ws.data.headers?.["sec-websocket-key"];
      
      // Try API key auth first
      const apiKeyContext = secWebSocketKey
        ? pendingApiKeyAuth.get(secWebSocketKey)
        : undefined;

      if (apiKeyContext) {
        // API key was validated in beforeHandle
        const connection: Connection = {
          id: connectionId,
          apiKey: apiKeyContext,
          projectId: apiKeyContext.projectId,
          channels: new Set(),
          connectedAt: new Date(),
          lastPing: new Date(),
          ws: ws.raw as ServerWebSocket<WebSocketData>,
          authType: "apiKey",
        };

        // Clean up pending auth
        if (secWebSocketKey) {
          pendingApiKeyAuth.delete(secWebSocketKey);
        }

        logger.info("WebSocket connection established (API key)", {
          connectionId,
          apiKeyId: apiKeyContext.apiKeyId,
          projectId: connection.projectId,
        });

        ws.send(
          JSON.stringify({
            type: "connected",
            connectionId,
            projectId: connection.projectId,
            authType: "apiKey",
            timestamp: Date.now(),
          }),
        );

        connections.set(connectionId, connection);
        wsInstances.set(connectionId, ws.raw as ServerWebSocket<WebSocketData>);
        return;
      }

      // Try session auth
      const user = secWebSocketKey
        ? pendingSessionAuth.get(secWebSocketKey)
        : undefined;

      if (user) {
        const connection: Connection = {
          id: connectionId,
          userId: user.id,
          projectId: projectId,
          channels: new Set(),
          connectedAt: new Date(),
          lastPing: new Date(),
          ws: ws.raw as ServerWebSocket<WebSocketData>,
          authType: "session",
        };

        // Clean up pending auth
        if (secWebSocketKey) {
          pendingSessionAuth.delete(secWebSocketKey);
        }

        logger.info("WebSocket connection established (session)", {
          connectionId,
          userId: user.id,
          projectId: connection.projectId,
        });

        ws.send(
          JSON.stringify({
            type: "connected",
            connectionId,
            userId: user.id,
            projectId: connection.projectId,
            authType: "session",
            timestamp: Date.now(),
          }),
        );

        connections.set(connectionId, connection);
        wsInstances.set(connectionId, ws.raw as ServerWebSocket<WebSocketData>);
      } else {
        // No valid auth found, close connection
        ws.close(1008, "Unauthorized: No valid authentication");
      }
    },
    message(ws, message) {
      const connection = connections.get(ws.id);
      if (!connection) {
        ws.close(1008, "Connection not found");
        return;
      }

      // Update last ping
      connection.lastPing = new Date();

      try {
        // Parse message
        const data =
          typeof message === "string" ? JSON.parse(message) : message;

        // Handle different message types
        switch (data.type) {
          case "ping":
            ws.send(
              JSON.stringify({
                type: "pong",
                timestamp: Date.now(),
              }),
            );
            break;

          case "subscribe":
            if (data.channel) {
              const channel = data.channel as string;
              connection.channels.add(channel);

              if (!channels.has(channel)) {
                channels.set(channel, new Set());
              }
              channels.get(channel)!.add(ws.id);

              ws.send(
                JSON.stringify({
                  type: "subscribed",
                  channel,
                  timestamp: Date.now(),
                }),
              );
            }
            break;

          case "unsubscribe":
            if (data.channel) {
              const channel = data.channel as string;
              connection.channels.delete(channel);

              const channelConnections = channels.get(channel);
              if (channelConnections) {
                channelConnections.delete(ws.id);
                if (channelConnections.size === 0) {
                  channels.delete(channel);
                }
              }

              ws.send(
                JSON.stringify({
                  type: "unsubscribed",
                  channel,
                  timestamp: Date.now(),
                }),
              );
            }
            break;

          case "publish":
            if (data.channel && data.message) {
              const channel = data.channel as string;
              const channelConnections = channels.get(channel);

              if (channelConnections) {
                // Broadcast to all connections in channel
                for (const connId of channelConnections) {
                  const conn = connections.get(connId);
                  if (conn) {
                    // Get WebSocket instance (this is simplified - in production, store ws instances)
                    // For now, we'll use Elysia's publish mechanism
                    ws.publish(channel, {
                      type: "message",
                      channel,
                      message: data.message,
                      sender: connection.userId,
                      timestamp: Date.now(),
                    });
                  }
                }
              }
            }
            break;

          default:
            ws.send(
              JSON.stringify({
                type: "error",
                message: `Unknown message type: ${data.type}`,
                timestamp: Date.now(),
              }),
            );
        }
      } catch (error) {
        ws.send(
          JSON.stringify({
            type: "error",
            message:
              error instanceof Error ? error.message : "Invalid message format",
            timestamp: Date.now(),
          }),
        );
      }
    },
    close(ws) {
      const connection = connections.get(ws.id);
      if (connection) {
        // Remove from all channels
        for (const channel of connection.channels) {
          const channelConnections = channels.get(channel);
          if (channelConnections) {
            channelConnections.delete(ws.id);
            if (channelConnections.size === 0) {
              channels.delete(channel);
            }
          }
        }
        connections.delete(ws.id);
        wsInstances.delete(ws.id); // Remove WebSocket instance
      }
    },
    // Error handler for WebSocket
    // @ts-expect-error - Elysia WebSocket error handler type signature mismatch
    // The runtime signature is correct (ws, error) but TypeScript expects a different type
    error(ws: { id: string; send: (data: string) => void }, error: unknown) {
      const errorMessage = error instanceof Error ? error.message : "Unknown error";
      logger.error("WebSocket error", error, {
        connectionId: ws.id,
      });
      try {
        ws.send(
          JSON.stringify({
            type: "error",
            message: errorMessage,
            timestamp: Date.now(),
          }),
        );
      } catch {
        // Ignore send errors if connection is already closed
      }
    },
  })
  // Broadcast message to channel
  .resolve(authResolver)
  .post(
    "/channels/:id/broadcast",
    async ({ user, params, body }) => {
      requireAuth(user);
      // TODO: Implement broadcast logic
      return {
        message: "Message broadcasted",
      };
    },
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
      }),
      body: t.Object({
        event: t.String(),
        data: t.Any(),
      }),
      response: {
        200: t.Object({
          message: t.String(),
        }),
      },
    },
  )
  // Get presence info for channel
  .get(
    "/channels/:id/presence",
    async ({ user, params }) => {
      requireAuth(user);
      const channel = channels.get(params.id);
      if (!channel) {
        throw new NotFoundError("Channel", params.id);
      }

      const channelConnections = Array.from(channel)
        .map((connId) => connections.get(connId))
        .filter((conn): conn is Connection => conn !== undefined);

      return {
        users: channelConnections
          .filter((conn) => conn.userId) // Only include connections with userId
          .map((conn) => ({
            id: conn.userId!,
            connectedAt: conn.connectedAt,
          })),
        count: channelConnections.length,
      };
    },
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          users: t.Array(
            t.Object({
              id: t.String(),
              connectedAt: t.Date(),
            }),
          ),
          count: t.Number(),
        }),
      },
    },
  )
  // Kick user from channel
  .post(
    "/channels/:id/kick",
    async ({ user, params, body }) => {
      requireAuth(user);
      // TODO: Implement kick logic
      return {
        message: "User kicked from channel",
      };
    },
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
      }),
      body: t.Object({
        userId: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          message: t.String(),
        }),
      },
    },
  )
  // List active connections (admin only)
  .get(
    "/connections",
    async ({ user, query }) => {
      requireAuth(user);
      // TODO: Check admin permissions

      const projectId = query.projectId as string | undefined;
      const connectionList = Array.from(connections.values())
        .filter((conn) => !projectId || conn.projectId === projectId)
        .map((conn) => ({
          id: conn.id,
          userId: conn.userId ?? "",
          projectId: conn.projectId ?? undefined,
          channels: Array.from(conn.channels),
          connectedAt: conn.connectedAt,
          lastPing: conn.lastPing,
        }));

      return {
        connections: connectionList,
        total: connectionList.length,
      };
    },
    {
      query: t.Object({
        projectId: t.Optional(t.String()),
      }),
      response: {
        200: t.Object({
          connections: t.Array(
            t.Object({
              id: t.String(),
              userId: t.String(),
              projectId: t.Optional(t.String()),
              channels: t.Array(t.String()),
              connectedAt: t.Date(),
              lastPing: t.Date(),
            }),
          ),
          total: t.Number(),
        }),
      },
    },
  )
  // Disconnect a client
  .delete(
    "/connections/:id",
    async ({ user, params }) => {
      requireAuth(user);
      // TODO: Check admin permissions or ownership

      const connection = connections.get(params.id);
      if (!connection) {
        return {
          error: {
            message: "Connection not found",
            code: "NOT_FOUND",
          },
        };
      }

      // Close WebSocket connection
      // Note: In production, you'd need to store ws instances to close them
      connections.delete(params.id);

      return {
        message: "Connection disconnected",
      };
    },
    {
      params: t.Object({
        id: t.String({ minLength: 1 }),
      }),
      response: {
        200: t.Object({
          message: t.String(),
        }),
        404: t.Object({
          error: t.Object({
            message: t.String(),
            code: t.String(),
          }),
        }),
      },
    },
  );
