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
   * Authentication is handled via cookies (Better Auth session)
   * userId is optional and deprecated - user is extracted from session on server
   */
  async connect(options?: {
    userId?: string; // Deprecated: kept for backward compatibility, not used
    projectId?: string;
    onMessage?: (message: RealtimeMessage) => void;
    onError?: (error: Error) => void;
    onClose?: () => void;
  }): Promise<WebSocket> {
    // If already connected, return existing connection
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return this.ws;
    }

    // Check if user is authenticated (session exists)
    const isAuthenticated = await this.checkAuthentication();
    if (!isAuthenticated) {
      throw new Error(
        "Not authenticated. Please sign in before connecting to realtime.",
      );
    }

    // Store options for reconnection
    this.connectOptions = {
      projectId: options?.projectId,
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
    // Ensure baseURL ends with / to properly append path
    const wsBase = wsBaseURL.endsWith("/") ? wsBaseURL : `${wsBaseURL}/`;
    const wsUrl = new URL("realtime", wsBase);

    // Add projectId to query if provided
    const projectId = this.connectOptions?.projectId || this.config.projectId;
    if (projectId) {
      wsUrl.searchParams.append("projectId", projectId);
    }

    const ws = new WebSocket(wsUrl.toString());
    ws.binaryType = "arraybuffer";

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
