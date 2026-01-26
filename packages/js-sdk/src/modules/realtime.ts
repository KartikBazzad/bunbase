/**
 * Realtime Module
 */

import type { BunBaseConfig, BunBaseClient } from "../client";
import type { RealtimeMessage, RealtimeChannel } from "../types";

export interface RealtimeModuleOptions {
  autoReconnect?: boolean;
  maxReconnectAttempts?: number;
  reconnectDelay?: number;
  onReconnecting?: () => void;
  onReconnected?: () => void;
}

export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "reconnecting";

export class RealtimeModule {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts: number;
  private reconnectDelay: number;
  private autoReconnect: boolean;
  private connectionState: ConnectionState = "disconnected";
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private connectOptions: {
    projectId?: string;
    apiKey?: string;
    onMessage?: (message: RealtimeMessage) => void;
    onError?: (error: Error) => void;
    onClose?: () => void;
  } | null = null;

  constructor(
    private config: BunBaseConfig,
    private client: BunBaseClient,
    private options?: RealtimeModuleOptions,
  ) {
    this.autoReconnect = options?.autoReconnect ?? true;
    this.maxReconnectAttempts = options?.maxReconnectAttempts ?? 5;
    this.reconnectDelay = options?.reconnectDelay ?? 1000;
  }

  /**
   * Get current connection state
   */
  getState(): ConnectionState {
    return this.connectionState;
  }

  /**
   * Check if user is authenticated before connecting
   */
  private async checkAuthentication(): Promise<boolean> {
    try {
      await this.client.auth.getUser();
      return true;
    } catch (error) {
      return false;
    }
  }

  /**
   * Connect to realtime WebSocket
   * Supports both session-based auth (cookies) and API key auth (headers)
   * userId is optional and deprecated - user is extracted from session on server
   */
  async connect(options?: {
    userId?: string; // Deprecated: kept for backward compatibility, not used
    projectId?: string;
    apiKey?: string; // API key for authentication (alternative to session)
    onMessage?: (message: RealtimeMessage) => void;
    onError?: (error: Error) => void;
    onClose?: () => void;
  }): Promise<WebSocket> {
    // If already connected, return existing connection
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return this.ws;
    }

    // Check authentication: either session or API key
    const hasApiKey = !!options?.apiKey || !!this.config.apiKey;
    const hasSession = await this.checkAuthentication();

    if (!hasSession && !hasApiKey) {
      throw new Error(
        "Not authenticated. Please sign in or provide an API key before connecting to realtime.",
      );
    }

    // Store options for reconnection
    this.connectOptions = {
      projectId: options?.projectId,
      apiKey: options?.apiKey || this.config.apiKey,
      onMessage: options?.onMessage,
      onError: options?.onError,
      onClose: options?.onClose,
    };

    return this._connect();
  }

  /**
   * Internal connect method (used for initial connection and reconnection)
   */
  private _connect(): WebSocket {
    if (
      this.connectionState === "connecting" ||
      this.connectionState === "reconnecting"
    ) {
      // Already connecting, return existing or wait
      if (this.ws) {
        return this.ws;
      }
    }

    this.connectionState =
      this.reconnectAttempts > 0 ? "reconnecting" : "connecting";

    if (this.reconnectAttempts > 0 && this.options?.onReconnecting) {
      this.options.onReconnecting();
    }

    const baseURL = this.config.baseURL || "http://localhost:3000/api";
    // WebSocket endpoint - cookies are automatically sent by browser
    // Convert http/https to ws/wss and construct WebSocket URL
    // The realtime route is mounted at /api/realtime (server prefix /api + route prefix /realtime)
    const wsBaseURL = baseURL.replace(/^http/, "ws").replace(/^https/, "wss");
    // Remove trailing slash if present, then append /realtime
    const cleanBase = wsBaseURL.replace(/\/$/, "");
    const wsUrl = new URL(`${cleanBase}/realtime`);

    // Add projectId to query if provided
    const projectId = this.connectOptions?.projectId || this.config.projectId;
    if (projectId) {
      wsUrl.searchParams.append("projectId", projectId);
    }

    // Prepare WebSocket connection with API key in headers if available
    const apiKey = this.connectOptions?.apiKey || this.config.apiKey;
    const wsOptions: any = {};

    // In Node.js/Bun environments, we can set headers
    // In browsers, headers are not supported, so API key must be sent after connection
    if (typeof process !== "undefined" && apiKey) {
      // Node.js/Bun environment - can set headers
      wsOptions.headers = {
        "X-API-Key": apiKey,
        Authorization: `Bearer ${apiKey}`,
      };
    }

    const ws = new WebSocket(wsUrl.toString(), wsOptions);
    ws.binaryType = "arraybuffer";

    // In browser environments, send API key as first message if needed
    if (
      typeof window !== "undefined" &&
      apiKey &&
      !this.connectOptions?.apiKey
    ) {
      ws.onopen = () => {
        // Send API key as first message (server will handle it)
        ws.send(
          JSON.stringify({
            type: "auth",
            apiKey: apiKey,
          }),
        );
      };
    }

    ws.onopen = () => {
      const wasReconnecting = this.reconnectAttempts > 0;
      this.reconnectAttempts = 0;
      this.connectionState = "connected";

      if (wasReconnecting && this.options?.onReconnected) {
        this.options.onReconnected();
      }

      console.log("Realtime connected");
    };

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as RealtimeMessage;

        // Handle connection confirmation
        if (message.type === "connected") {
          console.log("Realtime connection confirmed", message);
        }

        if (this.connectOptions?.onMessage) {
          this.connectOptions.onMessage(message);
        }
      } catch (error) {
        console.error("Failed to parse message:", error);
      }
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      this.connectionState = "disconnected";

      if (this.connectOptions?.onError) {
        this.connectOptions.onError(new Error("WebSocket connection error"));
      }
    };

    ws.onclose = (event) => {
      this.connectionState = "disconnected";
      console.log("Realtime disconnected", {
        code: event.code,
        reason: event.reason,
      });

      if (this.connectOptions?.onClose) {
        this.connectOptions.onClose();
      }

      // Handle authentication errors (401) - don't reconnect
      if (event.code === 1008 || event.code === 4001) {
        // Unauthorized or authentication failed
        console.error("Realtime connection closed: Authentication failed");
        this.reconnectAttempts = this.maxReconnectAttempts; // Stop reconnection attempts
        return;
      }

      // Attempt to reconnect if auto-reconnect is enabled
      if (
        this.autoReconnect &&
        this.reconnectAttempts < this.maxReconnectAttempts
      ) {
        this.reconnectAttempts++;
        const delay =
          this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

        this.reconnectTimer = setTimeout(() => {
          this._connect();
        }, delay);
      } else if (this.reconnectAttempts >= this.maxReconnectAttempts) {
        console.error("Realtime: Max reconnection attempts reached");
      }
    };

    this.ws = ws;
    return ws;
  }

  /**
   * Subscribe to a channel
   */
  subscribe(channel: string): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error("WebSocket not connected");
    }

    this.ws.send(
      JSON.stringify({
        type: "subscribe",
        channel,
      }),
    );
  }

  /**
   * Unsubscribe from a channel
   */
  unsubscribe(channel: string): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error("WebSocket not connected");
    }

    this.ws.send(
      JSON.stringify({
        type: "unsubscribe",
        channel,
      }),
    );
  }

  /**
   * Publish a message to a channel
   */
  publish(channel: string, message: any): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error("WebSocket not connected");
    }

    this.ws.send(
      JSON.stringify({
        type: "publish",
        channel,
        message,
      }),
    );
  }

  /**
   * Send a ping
   */
  ping(): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }

    this.ws.send(
      JSON.stringify({
        type: "ping",
      }),
    );
  }

  /**
   * Disconnect
   */
  disconnect(): void {
    // Cancel any pending reconnection
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    // Reset reconnection attempts
    this.reconnectAttempts = 0;
    this.connectionState = "disconnected";

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    // Clear stored options
    this.connectOptions = null;
  }

  /**
   * Manually trigger reconnection
   */
  reconnect(): Promise<WebSocket> {
    this.disconnect();
    this.reconnectAttempts = 0;
    return this.connect(this.connectOptions || undefined);
  }
}
