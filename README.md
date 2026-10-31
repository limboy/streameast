# StreamEast NBA

A lightweight, distraction-free desktop application for streaming NBA games, viewing live scores, and checking upcoming schedules. Built with [MyGo](https://github.com/egoist/mygo) (Go + native Webview) and TypeScript.

---

## Features

- **Multi-Source Streaming**:
  - Built-in provider support for **Streamseaste** (`streamseaste.cx`) and **StreamEast** mirrors (`v2.streameast.ga`, `v2.streameast.ch`, `v2.thestreameast.fun`).
  - Switch streaming domains instantly via the dropdown in the window title bar.
  - Automatic source speed testing with real-time latency measurements.

- **Live Games & Real-Time Scores**:
  - Instant game status display (quarters, halftime, overtime, live scores).
  - Automatically filters out completed games to focus on active and upcoming games for today.
  - High-resolution team logos with automatic light and dark mode adaptation.

- **Clean, Ad-Free Video Player**:
  - Watch games directly in an integrated player window.
  - Stream deobfuscation and popup/ad blocking.
  - Multi-stream switching (Stream 1, Stream 2, etc.) right from the player title bar.
  - Fullscreen support with auto-hiding navigation bars.

- **Upcoming Schedule**:
  - 3-day schedule with local game start times and TV broadcast networks.

- **Native macOS Experience**:
  - Seamless borderless title bar integrating with native macOS traffic light buttons.
  - Keyboard shortcuts (<kbd>⌘R</kbd> to refresh games and re-test source speeds).
  - Refresh button with hover tooltip displaying the exact last updated time.

---

## Tech Stack

- **Backend / Desktop Runtime**: [Go](https://go.dev/) (1.27+), [MyGo](https://github.com/egoist/mygo)
- **Frontend**: TypeScript, [Vite](https://vite.dev/), Modern CSS
- **Package Manager**: [Bun](https://bun.sh/) (or Node.js / npm)

---

## Getting Started

### Prerequisites

- [Go](https://go.dev/doc/install) 1.27 or higher
- [Bun](https://bun.sh/) (recommended) or [Node.js](https://nodejs.org/)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/streameast.git
   cd streameast
   ```

2. Install frontend dependencies:
   ```bash
   bun install
   # or: npm install
   ```

### Development

Run the app in development mode with live reload:

```bash
bun run dev
# or: npm run dev
```

This starts the Vite web dev server and launches the desktop app window.

To develop the web interface in a browser:

```bash
bun run dev:web
```

### Building the Desktop App

To compile a standalone desktop application binary:

```bash
bun run build
# or: npm run build
```

The output binary and macOS `.app` bundle will be generated in the `build/` directory.

### Running Tests

Run all unit and integration tests:

```bash
go test -v ./...
```

---

## Project Structure

```
├── main.go               # App entrypoint, window initialization, and settings management
├── provider.go           # Multi-provider abstraction (StreamEast & Streamseaste)
├── streamseaste.go       # Streamseaste scraper & livescore parser
├── streameast.go         # StreamEast scraper & game parser
├── player.go             # Player window controller & navigation restrictions
├── player.js             # Player page script (stream deobfuscation, server switcher)
├── logos.go              # Team naming and ESPN logo resolution
├── schedule.go           # ESPN 3-day schedule fetcher
├── index.html            # Main game list HTML layout
├── src/
│   ├── main.ts           # Frontend logic (game cards, source select, refresh action)
│   ├── mygo.ts           # Generated TypeScript bindings for Go functions
│   └── style.css         # Styling for title bar, game cards, and schedule
└── testdata/             # Test fixtures for parsing verification
```

---

## License

MIT
