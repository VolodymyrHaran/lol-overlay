# LoL Group Helper

A real-time League of Legends summoner spell tracker built with Go and React.

The application automatically reads the current Champion Select from the League Client (LCU), synchronizes players into rooms and tracks summoner spell cooldowns in real time using WebSockets.

## Features

- 🔗 League Client (LCU) integration
- 👥 Automatic room creation
- ⚔️ Champion Select synchronization
- ⏱️ Real-time summoner spell cooldown tracking
- 📡 WebSocket updates
- 🎮 React frontend
- 🎨 Tailwind CSS v4 + shadcn/ui
- ⚡ Fast Go backend

---

## Tech Stack

### Backend

- Go
- Gorilla WebSocket
- Riot LCU API
- REST API
- WebSockets

### Frontend

- React 19
- TypeScript
- Vite
- Tailwind CSS v4
- shadcn/ui

---

## Project Structure

```
.
├── cmd/
│   └── server/
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── services/
│   │   └── lib/
│   └── package.json
├── internal/
│   ├── handlers/
│   ├── models/
│   ├── services/
│   └── websocket/
└── go.mod
```

---

## Architecture

```
League Client (LCU)
        │
        ▼
 Go Backend
 ├── RoomService
 ├── LolClientService
 ├── ChampionService
 └── WebSocket Hub
        │
        ▼
 WebSocket
        │
        ▼
 React Frontend
 ├── Hooks
 ├── API Services
 └── UI Components
```

---

## Getting Started

### Backend

```bash
go mod tidy
go run ./cmd/server
```

Backend runs on

```
http://localhost:8080
```

---

### Frontend

```bash
cd frontend

npm install

npm run dev
```

Frontend runs on

```
http://localhost:5173
```

---

## API

### Get Room

```
GET /rooms/{roomId}
```

Returns current room state.

---

### Toggle Summoner Spell

```
POST /rooms/{roomId}/spells/toggle
```

Body

```json
{
    "gameName":"Player",
    "tagLine":"EUW",
    "spell":"Flash"
}
```

---

## WebSocket

Connect:

```
ws://localhost:8080/ws?roomId=<ROOM_ID>
```

The backend automatically broadcasts room updates whenever:

- Champion Select changes
- Summoner spell cooldown changes
- Players are synchronized

---

## Current Status

Implemented

- ✅ LCU connection
- ✅ Champion Select synchronization
- ✅ Automatic room creation
- ✅ Room cleanup
- ✅ WebSocket broadcasting
- ✅ Real-time cooldown tracking
- ✅ React frontend
- ✅ Tailwind CSS v4
- ✅ shadcn/ui

Planned

- Champion icons from Riot Data Dragon
- Summoner spell icons
- Blue / Red team separation
- In-game synchronization
- Overlay mode

---

## Screenshots

Coming soon.
