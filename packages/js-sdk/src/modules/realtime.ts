/**
 * Realtime Module
 */

import type { BunBaseConfig, BunBaseClient } from "../client";
import type { RealtimeMessage, RealtimeChannel } from "../types";

export interface RealtimeModuleOptions {
  // Additional realtime-specific options
}

export class RealtimeModule {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  constructor(
    private config: BunBaseConfig,
    client: BunBaseClient,
    private options?: RealtimeModuleOptions,
  ) {}

  /**
   * Connect to realtime WebSocket
   */
  connect(options?: {
    userId: string;
    projectId?: string;
    onMessage?: (message: RealtimeMessage) => void;
    onError?: (error: Error) => void;
    onClose?: () => void;
  }): WebSocket {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return this.ws;
    }

    const wsUrl = new URL("/realtime/ws", this.config.baseURL.replace("http", "ws"));
    if (options?.userId) {
      wsUrl.searchParams.append("userId", options.userId);
    }
    if (options?.projectId || this.config.projectId) {
      wsUrl.searchParams.append("projectId", options?.projectId || this.config.projectId);
    }

    const ws = new WebSocket(wsUrl.toString());
    ws.binaryType = "arraybuffer";

    ws.onopen = () => {
      this.reconnectAttempts = 0;
      console.log("Realtime connected");
    };

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as RealtimeMessage;
        if (options?.onMessage) {
          options.onMessage(message);
        }
      } catch (error) {
        console.error("Failed to parse message:", error);
      }
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      if (options?.onError) {
        options.onError(new Error("WebSocket connection error"));
      }
    };

    ws.onclose = () => {
      console.log("Realtime disconnected");
      if (options?.onClose) {
        options.onClose();
      }

      // Attempt to reconnect
      if (this.reconnectAttempts < this.maxReconnectAttempts) {
        this.reconnectAttempts++;
        setTimeout(() => {
          this.connect(options);
        }, this.reconnectDelay * Math.pow(2, this.reconnectAttempts));
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
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}
