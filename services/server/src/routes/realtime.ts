import { Elysia, t } from "elysia";
import { authResolver } from "../middleware/auth";
import { requireAuth } from "../lib/auth-helpers";
import { UnauthorizedError, NotFoundError } from "../lib/errors";
import { logger } from "../lib/logger";
import { auth } from "../auth";

// Connection management
interface Connection {
  id: string;
  userId: string;
  projectId?: string;
  channels: Set<string>;
  connectedAt: Date;
  lastPing: Date;
}

const connections = new Map<string, Connection>();
const channels = new Map<string, Set<string>>(); // channel -> connection IDs

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

export const realtimeRoutes = new Elysia({ prefix: "/realtime" })
  .resolve(authResolver)
  .ws("/", {
    // Authenticate connection using Better Auth session from cookies
    beforeHandle: async ({ headers, status }) => {
      try {
        // Extract session from cookies via Better Auth
        // WebSocket upgrade requests include cookies in the Cookie header
        const session = await auth.api.getSession({ headers });

        if (!session?.user) {
          return status(401, {
            error: {
              message: "Unauthorized: No valid session found",
              code: "UNAUTHORIZED",
            },
          });
        }

        // Session is valid, user will be available in open handler via authResolver
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
    open(ws, { user }) {
      // User is available from authResolver (extracted from session)
      if (!user) {
        ws.close(1008, "Unauthorized: No user found");
        return;
      }

      const connectionId = ws.id;

      // Extract projectId from query params if provided
      const projectId = ws.data.query?.projectId as string | undefined;

      const connection: Connection = {
        id: connectionId,
        userId: user.id, // Use authenticated user from session
        projectId,
        channels: new Set(),
        connectedAt: new Date(),
        lastPing: new Date(),
      };

      connections.set(connectionId, connection);

      logger.info("WebSocket connection established", {
        connectionId,
        userId: user.id,
        projectId,
      });

      // Send welcome message
      ws.send(
        JSON.stringify({
          type: "connected",
          connectionId,
          userId: user.id,
          timestamp: Date.now(),
        }),
      );
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
      }
    },
    error(ws, error) {
      logger.error("WebSocket error", error, {
        connectionId: ws.id,
      });
      ws.send(
        JSON.stringify({
          type: "error",
          message: error instanceof Error ? error.message : "Unknown error",
          timestamp: Date.now(),
        }),
      );
    },
  })
  // Broadcast message to channel
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
        users: channelConnections.map((conn) => ({
          id: conn.userId,
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
          userId: conn.userId,
          projectId: conn.projectId,
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
