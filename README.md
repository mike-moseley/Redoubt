# Redoubt

[![CI](https://github.com/mike-moseley/Redoubt/actions/workflows/ci.yml/badge.svg)](https://github.com/mike-moseley/Redoubt/actions/workflows/ci.yml)

A multiplayer terminal roguelike/city-builder with MUD-like combat, written in Go.

## Overview

Two binaries: a headless game server and a terminal client. The server owns all game state authoritatively. Clients send intents, receive field-of-view snapshots, and render them via [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Architecture

### Server

Runs a tick-based simulation at ~6.7 ticks per second (150ms intervals). Each tick:

1. Drains all player commands queued since the last tick (one command per player per tick)
2. Resolves commands and advances time-driven systems (combat, construction, NPC movement)
3. Computes each player's field of view and sends a snapshot to their client

Concurrency is handled via channels — a single game loop goroutine owns all world state. Connected clients each run in their own goroutine and communicate with the game loop through a shared inbound channel. The game loop writes responses to per-session outbound channels. No mutexes needed — only the game loop goroutine ever touches the ECS store.

Four event types flow through the inbound channel: tick events, connect/disconnect events, and `protocol.ClientEnvelope` (wrapping all player commands). All implement a marker interface so the channel is typed and the game loop dispatches via type switch. Inside a `ClientEnvelope` case, an inner switch on `Type` handles `TypeMove` and `TypeChatMessage`.

### Client

Runs a Bubble Tea TUI. Holds a local snapshot of the last FOV update received from the server. Key presses are translated to intents and sent to the server — the client never mutates game state directly. Bubble Tea handles terminal diffing internally so the client re-renders the full snapshot each frame.

Two panes rendered with Lip Gloss borders: a game viewport (map tiles + entities) and a chat area (`bubbles/viewport` for scroll history + `bubbles/textinput` for input). Active pane is highlighted green, inactive gray. Modal input: Space enters chat mode, Esc exits.

On connect, the client immediately sends a `ConnectMessage` with the player name before the normal message loop begins. Flags: `--name`, `--address`.

### ECS (Entity Component System)

World state is managed via ECS. Entities are `uint64` IDs. Components are plain data structs. The `Store` in `internal/core` holds one `map[EntityID]ComponentType` per component type — fixed named fields, not dynamic. Systems iterate component maps directly; querying entities with multiple components uses a generic iterator function `Query[A, B](mapA, mapB)` that iterates the smaller map and checks the larger.

Initial components: `Position` (int32 x/y), `Renderable` (rune, color string, type ID), `Name` (string).

### Data

Entity types (mobs, buildings, items) are data-driven and loaded from TOML files at server startup. Each entity type has a **definition** (static, loaded once, shared) and **instances** (dynamic ECS components, mutate during play). Definitions live in `data/`. A shared TOML loader utility in `core` handles file decoding; each domain package owns its own definition types.

### FOV Snapshots

Each tick the server computes what each player can see and sends a `protocol.FOVSnapshot` containing:
- Tiles — `[]TileType` in row-major order (`index = y * width + x`), plus viewport width/height
- Entities — `[]RenderableEntity` pairing `Position` and `Renderable` for each visible entity
- Player position — viewport anchor

The client maintains a `StaleMap` — previously seen tiles rendered in a dim color when they fall outside the current FOV. Full snapshot every tick; no dirty-cell tracking needed since Bubble Tea diffs terminal output internally.

## Package Structure

```
cmd/
  server/        # server entry point, game loop, command dispatch
  client/        # client entry point

internal/
  core/          # shared types: Event interface, Session, EntityID, TileType, ECS Store, components
  netcode/       # TCP listener, per-connection goroutines, ConnectMessage handshake
  bubble/        # Bubble Tea model (bubble.go), key handlers (input.go), renderer (render.go), helpers (helpers.go), constants (consts.go)
  world/         # grid, tiles, FOV calculation
  entity/        # mob/NPC definitions and TOML loader
  combat/        # initiative queue, damage resolution, status effect definitions
  city/          # buildings, resources, construction queues, building definitions
  protocol/      # wire types: clientMessage.go, serverMessage.go, chat.go, consts.go

data/
  mobs.toml
  buildings.toml
  items.toml
```

### Import Graph

`core` imports nothing internal. All other packages import `core`. Additional dependencies:

- `protocol` → `core` (needs Session, Position, TileType, RenderableEntity)
- `netcode` → `core`, `protocol`
- `bubble` → `protocol`, `core`
- `cmd/server` → everything
- `cmd/client` → `bubble`, `protocol`, `core`

Domain packages (`world`, `entity`, `combat`, `city`) import `core` only — no cross-dependencies between them. All cross-domain interaction happens through the game loop in `cmd/server`.

## Key Design Decisions

- **Server-authoritative**: clients are views, not participants in simulation
- **Channels over mutexes**: single game loop goroutine owns all ECS state; no locking needed
- **ECS for performance**: supports dense NPC populations in cities and wilderness without pointer chasing
- **Tile type enums**: server sends type IDs; client resolves glyphs and colors locally
- **Entity renderables**: server sends glyph/color/typeID directly — client doesn't maintain entity lookup tables
- **FOV-only snapshots**: server sends only what each player can see, row-major tile array + entity list
- **Definition/instance split**: static game data in TOML, dynamic world state in ECS
- **Two-binary architecture**: server is headless; Bubble Tea runs only on the client
- **JSON wire format**: used for client-server communication to support non-Go clients in the future. A planned learning exercise is to replace this with a custom binary encoding protocol.
