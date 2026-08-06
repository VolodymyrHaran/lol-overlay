import type { Room } from "../types";

export interface CurrentRoomMessage {
  type: "current_room";
  roomId: string;
}

export interface RoomUpdateMessage {
  type: "room_update";
  room: Room;
}

export type WebSocketMessage =
  | CurrentRoomMessage
  | RoomUpdateMessage;