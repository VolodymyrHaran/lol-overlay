import type { CurrentRoomResponse } from "../types/currentRoom";

const API_BASE_URL =
  import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export async function getCurrentRoom(): Promise<CurrentRoomResponse | null> {
  const response = await fetch(`${API_BASE_URL}/current-room`);

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    throw new Error(
      `Failed to get current room: ${response.status} ${response.statusText}`,
    );
  }

  return (await response.json()) as CurrentRoomResponse;
}