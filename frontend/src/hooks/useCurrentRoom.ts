import { useEffect, useState } from "react";

import type {
  CurrentRoomMessage,
} from "../types/websocket";

interface UseCurrentRoomResult {
  roomId: string | null;
  isLoading: boolean;
  error: string | null;
}

import {
  WS_BASE_URL,
} from "../config/backend";

const INITIAL_RECONNECT_DELAY_MS = 1_000;
const MAX_RECONNECT_DELAY_MS = 10_000;

export function useCurrentRoom(): UseCurrentRoomResult {
  const [roomId, setRoomId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let disposed = false;
    let socket: WebSocket | null = null;
    let reconnectTimer: number | null = null;
    let reconnectAttempt = 0;

function connect(): void {
  if (
    disposed ||
    socket?.readyState === WebSocket.OPEN ||
    socket?.readyState === WebSocket.CONNECTING
  ) {
    return;
  }

  socket = new WebSocket(
    `${WS_BASE_URL}/ws/current-room`,
  );

  socket.onopen = () => {
    if (disposed) {
      return;
    }

    reconnectAttempt = 0;
    setError(null);
  };

  socket.onmessage = (
  event: MessageEvent<string>,
) => {
  console.log(
    "Current-room message received:",
    event.data,
  );

  if (disposed) {
    return;
  }

  try {
    const message = JSON.parse(
      event.data,
    ) as CurrentRoomMessage;

    if (message.type !== "current_room") {
      console.warn(
        "Unexpected current-room message:",
        message,
      );
      return;
    }

    console.log(
      "Changing room ID to:",
      message.roomId,
    );

    setRoomId(message.roomId || null);
    setIsLoading(false);
    setError(null);
  } catch {
    setError(
      "Failed to parse current-room update",
    );
    setIsLoading(false);
  }
};

  socket.onerror = (event) => {
    console.error(
        "Current-room WebSocket error",
        event,
    );
    };

  socket.onclose = (event: CloseEvent) => {
    console.log("Current-room WebSocket closed", {
        code: event.code,
        reason: event.reason,
        wasClean: event.wasClean,
    });

    if (disposed) {
        return;
    }

    socket = null;

    const delay = Math.min(
        INITIAL_RECONNECT_DELAY_MS *
        2 ** reconnectAttempt,
        MAX_RECONNECT_DELAY_MS,
    );

    reconnectAttempt += 1;

    reconnectTimer = window.setTimeout(
        connect,
        delay,
    );
    };
}

    connect();

    return () => {
      disposed = true;

      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
      }

      socket?.close();
    };
  }, []);

  return {
    roomId,
    isLoading,
    error,
  };
}