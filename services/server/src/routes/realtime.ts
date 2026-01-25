import { Elysia, t } from "elysia";
import { authResolver } from "../middleware/auth";
import { requireAuth } from "../lib/auth-helpers";
import { UnauthorizedError } from "../lib/errors";

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
  .ws("/ws", {
    // Authenticate connection
    beforeHandle({ headers, status }) {
      // Extract token from headers or query
      const token = headers.authorization?.replace("Bearer ", "") ||
        headers["sec-websocket-protocol"]?.split(",")[0]?.trim();
      
      if (!token) {
        return status(401, "Unauthorized: Missing authentication token");
      }

      // TODO: Verify token and get user
      // For now, we'll handle auth in the open handler
    },
    open(ws) {
      // Extract user from query parameters in the WebSocket URL
      // Elysia WebSocket query params are available via ws.data
      // For now, we'll use a simplified approach - in production, extract from URL
      const connectionId = ws.id;
      
      // TODO: Extract userId and projectId from WebSocket connection query params
      // For now, create connection without user validation (should be added)
      const connection: Connection = {
        id: connectionId,
        userId: "anonymous", // TODO: Extract from auth token
        projectId: undefined,
        channels: new Set(),
        connectedAt: new Date(),
        lastPing: new Date(),
      };

      connections.set(connectionId, connection);

      // Send welcome message
      ws.send(JSON.stringify({
        type: "connected",
        connectionId,
        timestamp: Date.now(),
      }));
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
        const data = typeof message === "string" ? JSON.parse(message) : message;

        // Handle different message types
        switch (data.type) {
          case "ping":
            ws.send(JSON.stringify({
              type: "pong",
              timestamp: Date.now(),
            }));
            break;

          case "subscribe":
            if (data.channel) {
              const channel = data.channel as string;
              connection.channels.add(channel);

              if (!channels.has(channel)) {
                channels.set(channel, new Set());
              }
              channels.get(channel)!.add(ws.id);

              ws.send(JSON.stringify({
                type: "subscribed",
                channel,
                timestamp: Date.now(),
              }));
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

              ws.send(JSON.stringify({
                type: "unsubscribed",
                channel,
                timestamp: Date.now(),
              }));
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
            ws.send(JSON.stringify({
              type: "error",
              message: `Unknown message type: ${data.type}`,
              timestamp: Date.now(),
            }));
        }
      } catch (error) {
        ws.send(JSON.stringify({
          type: "error",
          message: error instanceof Error ? error.message : "Invalid message format",
          timestamp: Date.now(),
        }));
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
      console.error("WebSocket error:", error);
      ws.send(JSON.stringify({
        type: "error",
        message: error instanceof Error ? error.message : "Unknown error",
        timestamp: Date.now(),
      }));
    },
  })
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
