# Dispense Dashboard

A React TypeScript web application for managing Dispense sandboxes and tasks through a modern web interface.

## 🎯 Features

- **Sandbox Management**: Create, delete, start/stop sandboxes with real-time status updates
- **Task Execution**: Run Claude tasks with streaming logs and proper formatting
- **File Monitoring**: View modified files with status indicators
- **Terminal Interface**: WebSocket-based terminal with command history
- **Responsive Design**: Modern UI with three-panel layout

## 🚀 Quick Start

### Development

1. **Start the dashboard in development mode:**
   ```bash
   yarn nx serve dashboard
   ```
   Dashboard will be available at http://localhost:4200

2. **Start the Dispense server:**
   ```bash
   cd dispense && go run ./cmd/*.go server
   ```

### Production Build

1. **Build dashboard and embed in Go binary:**
   ```bash
   yarn build:dashboard
   ```

2. **Build complete Dispense binary with embedded dashboard:**
   ```bash
   yarn build:full
   ```

3. **Run the server with embedded dashboard:**
   ```bash
   cd dispense && ./dispense server
   ```

   The dashboard will be available at http://localhost:8081 (root path)
   API endpoints will be available at http://localhost:8081/api

## 🏗️ Architecture

### Technology Stack
- **Frontend**: React 18 + TypeScript
- **Build Tool**: Webpack (via Nx)
- **State Management**: React Context API
- **Styling**: CSS Modules with utility classes
- **HTTP Client**: Custom API wrapper around `@api-client-ts`
- **Real-time**: WebSocket and Server-Sent Events
- **Embedding**: Go `embed` package

### Project Structure
```
apps/dashboard/
├── src/
│   ├── components/          # React components
│   │   ├── layout/         # Layout components
│   │   ├── sandbox/        # Sandbox management
│   │   ├── task/           # Task execution
│   │   ├── files/          # File monitoring
│   │   └── terminal/       # Terminal interface
│   ├── contexts/           # React Context for state
│   ├── hooks/              # Custom React hooks
│   ├── services/           # API and WebSocket services
│   ├── types/              # TypeScript interfaces
│   ├── utils/              # Utilities and formatters
│   └── styles/             # Global CSS styles
└── dispense/internal/dashboard/
    ├── embed.go            # Go embedding logic
    └── static/             # Built dashboard assets
```

## 🔧 Development

### Available Commands

```bash
# Development server with hot reload
yarn nx serve dashboard

# Production build
yarn nx build dashboard

# Build dashboard and copy for embedding
yarn build:dashboard

# Full build (dashboard + Go binary)
yarn build:full

# Run tests
yarn nx test dashboard

# Run linting
yarn nx lint dashboard
```

### API Integration

The dashboard communicates with the Dispense server through:

- **REST API**: HTTP gateway at `/api/v1/*` endpoints
- **WebSocket**: Real-time terminal connections
- **Server-Sent Events**: Streaming task logs

### Real-time Features

- **Sandbox Status**: Polling every 5 seconds
- **Task Logs**: Server-Sent Events streaming
- **File Changes**: Polling every 10 seconds
- **Terminal**: WebSocket bidirectional communication

## 📊 Dashboard Layout

### Left Sidebar (300px)
- Project selector dropdown
- Sandbox list with status indicators
- Create/delete/start/stop actions
- Real-time status updates

### Center Panel (Flexible)
- **Top**: Streaming task logs with filtering
- **Bottom**: Task prompt with history

### Right Panel (250px)
- **Top**: Modified files list
- **Bottom**: Resizable terminal (200px default)

## 🔒 Security

- **CSP Headers**: Content Security Policy configured
- **CORS**: Proper cross-origin handling
- **Input Validation**: Sanitized user inputs
- **Authentication**: API key support (when configured)

## 🚀 Deployment

The dashboard is embedded directly into the Go binary:

1. **Build Process**: `yarn build:dashboard`
   - Builds optimized React app
   - Copies assets to `dispense/internal/dashboard/static/`

2. **Go Embedding**: `//go:embed static/*`
   - Files embedded at compile time
   - Served via HTTP handler at `/` (root)

3. **Server Integration**:
   - Routes `/` (root) to embedded dashboard files
   - Routes `/api/*` to gRPC gateway
   - Single binary deployment

## 📝 Configuration

### Environment Variables
- `DISPENSE_API_KEY`: API key for authentication
- `NODE_ENV`: Set to `production` for optimized builds

### Build Options
- **Base Path**: `/` (root path, configured in webpack)
- **Bundle Size**: ~837KB total (within spec requirements)
- **Caching**: Static assets cached for 1 year

## 🐛 Troubleshooting

### Build Issues
- Ensure dashboard is built before Go compilation
- Check that static files exist in `dispense/internal/dashboard/static/`

### Runtime Issues
- Verify server is running on correct ports
- Check browser console for JavaScript errors
- Ensure WebSocket connections are allowed

### Development
- Dashboard dev server: http://localhost:4200
- API endpoints: http://localhost:8081/api/v1/*
- WebSocket: ws://localhost:8081/ws/*

## 📈 Performance

- **Initial Load**: < 3 seconds
- **Bundle Size**: ~500KB gzipped
- **Memory Usage**: < 50MB
- **Real-time Latency**: < 100ms for task logs

## 🔮 Future Enhancements

- File editor integration
- Real-time collaboration
- Custom themes
- Advanced terminal features
- Mobile responsive improvements