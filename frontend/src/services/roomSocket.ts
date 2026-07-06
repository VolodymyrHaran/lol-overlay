import type { Room } from "../types";

export function connectToRoom(
  roomId: string,
  onRoomChanged: (room: Room) => void
) {
  const socket = new WebSocket(
    `ws://localhost:8080/ws?roomId=${roomId}`
  );

  socket.onmessage = (event) => {
    const data = JSON.parse(event.data) as Room;
    onRoomChanged(data);
  };

  return () => {
    socket.close();
  };
}