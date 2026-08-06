import PlayerCard from "./components/PlayerCard";
import { useCurrentRoom } from "./hooks/useCurrentRoom";
import { useRoom } from "./hooks/useRoom";
import ConnectionBadge from "./components/ConnectionBadge";

function App() {
  const {
    roomId,
    isLoading: isCurrentRoomLoading,
    error: currentRoomError,
  } = useCurrentRoom();

  const {
    room,
    isLoading: isRoomLoading,
    error: roomError,
    toggleSpell,
    connectionStatus,
  } = useRoom(roomId);

  if (isCurrentRoomLoading) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-zinc-950 text-zinc-100">
        <p>Searching for the current League room...</p>
      </main>
    );
  }

  if (currentRoomError) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-zinc-950 text-red-400">
        <p>{currentRoomError}</p>
      </main>
    );
  }

  if (!roomId) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-zinc-950 text-zinc-100">
        <div className="text-center">
          <h1 className="mb-2 text-2xl font-semibold">
            Waiting for Champion Select
          </h1>

          <p className="text-zinc-400">
            Start League of Legends and enter Champion Select.
          </p>
        </div>
      </main>
    );
  }

  if (isRoomLoading) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-zinc-950 text-zinc-100">
        <p>Loading room {roomId}...</p>
      </main>
    );
  }

  if (roomError && !room) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-zinc-950 text-red-400">
        <p>{roomError}</p>
      </main>
    );
  }

  if (!room) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-zinc-950 text-zinc-100">
        <p>Room not found.</p>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-zinc-950 p-6 text-zinc-100">
      <div className="mx-auto max-w-6xl">
        <header className="mb-6">
          <h1 className="text-3xl font-bold">LoL Timer</h1>

          <p className="mt-1 text-sm text-zinc-400">
            Room: {room.id}
          </p>
          <ConnectionBadge status={connectionStatus} />
        </header>

        <section className="grid gap-4">
          {room.players.map((player) => (
            <PlayerCard
              key={`${player.gameName}-${player.tagLine}`}
              player={player}
              onToggleSpell={toggleSpell}
            />
          ))}
        </section>
      </div>
    </main>
  );
}

export default App;