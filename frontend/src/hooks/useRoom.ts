import { useEffect, useState } from "react";
import type { Room } from "../types";
import { connectToRoom } from "../services/roomSocket";

export function useRoom(roomId: string) {
  const [room, setRoom] = useState<Room | null>(null);

  useEffect(() => {
    if (!roomId) {
      setRoom(null);
      return;
    }

    return connectToRoom(roomId, setRoom);
  }, [roomId]);

  return room;
}