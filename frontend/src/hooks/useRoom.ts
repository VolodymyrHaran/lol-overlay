import {
  useCallback,
  useEffect,
  useState,
} from "react";

import {
  API_BASE_URL,
  WS_BASE_URL,
} from "../config/backend";

import type {
  RoomUpdateMessage,
} from "../types/websocket";

import type { Room } from "../types";

const MAX_RECONNECT_DELAY_MS = 10_000;
const INITIAL_RECONNECT_DELAY_MS = 1_000;

export type ConnectionStatus =
  | "connecting"
  | "connected"
  | "reconnecting"
  | "disconnected";

interface UseRoomResult {
  room: Room | null;
  isLoading: boolean;
  error: string | null;
  connectionStatus: ConnectionStatus;
  toggleSpell: (
    gameName: string,
    tagLine: string,
    spell: string,
  ) => Promise<void>;
}

export function useRoom(
  roomId: string | null,
): UseRoomResult {
  const [room, setRoom] = useState<Room | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [
    connectionStatus,
    setConnectionStatus,
  ] = useState<ConnectionStatus>("disconnected");

  useEffect(() => {
    if (!roomId) {
      setRoom(null);
      setIsLoading(false);
      setError(null);
      setConnectionStatus("disconnected");
      return;
    }

    const activeRoomId = roomId;

    let isDisposed = false;
    let socket: WebSocket | null = null;
    let reconnectTimeoutId: number | null = null;
    let reconnectAttempt = 0;

    setRoom(null);
    setIsLoading(true);

    function scheduleReconnect(): void {
      if (isDisposed) {
        return;
      }

      const delay = Math.min(
        INITIAL_RECONNECT_DELAY_MS *
          2 ** reconnectAttempt,
        MAX_RECONNECT_DELAY_MS,
      );

      reconnectAttempt += 1;

      setConnectionStatus("reconnecting");

      reconnectTimeoutId = window.setTimeout(() => {
        connectWebSocket();
      }, delay);
    }

    function connectWebSocket(): void {
      if (isDisposed) {
        return;
      }

      setError(null);

      setConnectionStatus(
        reconnectAttempt === 0
          ? "connecting"
          : "reconnecting",
      );

      socket = new WebSocket(
        `${WS_BASE_URL}/ws?roomId=${encodeURIComponent(
          activeRoomId,
        )}`,
      );

      socket.onopen = () => {
        if (isDisposed) {
          return;
        }

        reconnectAttempt = 0;
        setConnectionStatus("connected");
        setError(null);
      };

      socket.onmessage = (
        event: MessageEvent<string>,
      ) => {
        if (isDisposed) {
          return;
        }

        try {
          const message = JSON.parse(
            event.data,
          ) as RoomUpdateMessage;

          if (message.type !== "room_update") {
            return;
          }

          setRoom(message.room);
          setIsLoading(false);
          setError(null);
        } catch {
          setError(
            "Failed to parse WebSocket room update",
          );
          setIsLoading(false);
        }
      };

      socket.onerror = () => {
        if (isDisposed) {
          return;
        }

        setConnectionStatus("reconnecting");

        // onclose запустит следующую попытку.
        socket?.close();
      };

      socket.onclose = () => {
        if (isDisposed) {
          return;
        }

        socket = null;
        scheduleReconnect();
      };
    }

    connectWebSocket();

    return () => {
      isDisposed = true;

      if (reconnectTimeoutId !== null) {
        window.clearTimeout(reconnectTimeoutId);
      }

      socket?.close();
    };
  }, [roomId]);

  const toggleSpell = useCallback(
    async (
      gameName: string,
      tagLine: string,
      spell: string,
    ): Promise<void> => {
      if (!roomId) {
        setError("Current room is not available");
        return;
      }

      try {
        const response = await fetch(
          `${API_BASE_URL}/rooms/${encodeURIComponent(
            roomId,
          )}/spells/toggle`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              gameName,
              tagLine,
              spell,
            }),
          },
        );

        if (!response.ok) {
          throw new Error(
            `Failed to toggle spell: ${response.status} ${response.statusText}`,
          );
        }

        setError(null);
      } catch (caughtError) {
        setError(
          caughtError instanceof Error
            ? caughtError.message
            : "Unknown spell toggle error",
        );
      }
    },
    [roomId],
  );

  return {
    room,
    isLoading,
    error,
    connectionStatus,
    toggleSpell,
  };
}