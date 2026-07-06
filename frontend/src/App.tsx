import { useState } from "react";
import { toggleSpell } from "./services/roomApi";
import PlayerCard from "./components/PlayerCard";
import { useRoom } from "./hooks/useRoom";

function App() {
  const [roomId, setRoomId] = useState("");

  const room = useRoom(roomId);
  return (
    <div className="min-h-screen bg-gray-950 text-white">
      <main className="mx-auto max-w-5xl px-6 py-8">
        <h1 className="mb-6 text-4xl font-bold">
          LoL Timer
        </h1>

        <input
          className="mb-8 w-full rounded-lg border border-gray-700 bg-gray-900 px-4 py-3 text-white outline-none focus:border-blue-500"
          value={roomId}
          onChange={(e) => setRoomId(e.target.value)}
          placeholder="Room ID"
        />

        {room && (
          <div>
            <h2 className="mb-4 text-xl text-gray-300">
              Room: {room.id}
            </h2>

            {room.players.map((player) => (
              <PlayerCard
                key={`${player.gameName}-${player.tagLine}`}
                player={player}
                onToggleSpell={(gameName, tagLine, spell) =>
                  toggleSpell(roomId, gameName, tagLine, spell)
                }
              />
            ))}
          </div>
        )}
      </main>
    </div>
  );
}

export default App;