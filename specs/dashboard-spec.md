# Dashboard Project Specification

## Overview

The Dashboard project is a TypeScript React web application that provides a web-based interface for managing Dispense sandboxes and tasks. The application will be embedded into the Dispense binary using Go's `embed` package and served alongside the gRPC and gRPC Gateway when running in server mode.

## Project Architecture

### Technology Stack
- **Framework**: React 18+ with TypeScript
- **Build Tool**: Webpack (via Nx)
- **State Management**: React Context API or Zustand
- **Styling**: CSS Modules or Styled Components
- **HTTP Client**: Custom DispenseClient from `@api-client-ts`
- **Embedding**: Go `embed` package for static assets

### Project Structure
```
apps/dashboard/
├── src/
│   ├── components/
│   │   ├── layout/
│   │   │   ├── DashboardLayout.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   ├── MainContent.tsx
│   │   │   └── RightPanel.tsx
│   │   ├── sandbox/
│   │   │   ├── SandboxList.tsx
│   │   │   ├── SandboxItem.tsx
│   │   │   └── SandboxSelector.tsx
│   │   ├── task/
│   │   │   ├── TaskLogs.tsx
│   │   │   ├── TaskPrompt.tsx
│   │   │   └── TaskStatus.tsx
│   │   ├── files/
│   │   │   ├── ModifiedFilesList.tsx
│   │   │   └── FileItem.tsx
│   │   └── terminal/
│   │       ├── Terminal.tsx
│   │       └── TerminalOutput.tsx
│   ├── hooks/
│   │   ├── useSandboxes.ts
│   │   ├── useTasks.ts
│   │   ├── useFiles.ts
│   │   └── useTerminal.ts
│   ├── services/
│   │   ├── api.ts
│   │   └── websocket.ts
│   ├── types/
│   │   ├── sandbox.ts
│   │   ├── task.ts
│   │   └── file.ts
│   ├── utils/
│   │   ├── formatters.ts
│   │   └── constants.ts
│   ├── styles/
│   │   ├── globals.css
│   │   ├── components/
│   │   └── layout/
│   ├── App.tsx
│   └── main.tsx
├── public/
│   └── index.html
├── project.json
├── tsconfig.json
└── webpack.config.js
```

## UI Layout Specification

### Layout Structure
The dashboard follows a three-panel layout as shown in the reference image:

#### Left Sidebar (Sandbox Management)
- **Width**: 300px (fixed)
- **Content**: 
  - Project selector dropdown
  - Sandbox list with scrollable container
  - Each sandbox item shows:
    - Sandbox name
    - Type (Local/Remote) with icon
    - Status indicator (Running/Stopped/Error)
    - Creation timestamp
    - Quick actions (Start/Stop/Delete)

#### Center Panel (Task Management)
- **Content**:
  - **Top Section**: Task logs view (scrollable)
    - Real-time streaming of task output
    - Color-coded log levels (STDOUT/STDERR/STATUS/ERROR)
    - Timestamp for each log entry
    - Auto-scroll to latest entry option
  - **Bottom Section**: Task prompt input
    - Multi-line textarea for task descriptions
    - Send button
    - Clear button
    - Character count indicator

#### Right Panel (File Management & Terminal)
- **Top Section**: Modified Files List
  - **Width**: 250px
  - **Content**:
    - List of modified files in current sandbox
    - File status indicators (Added/Modified/Deleted)
    - Click to open file in editor (future enhancement)
    - Refresh button
- **Bottom Section**: Terminal
  - **Height**: 200px (resizable)
  - **Content**:
    - Terminal output display
    - Command input field
    - Terminal tabs support
    - Clear terminal button

## Component Specifications

### 1. SandboxList Component
```typescript
interface SandboxListProps {
  projectId: string;
  onSandboxSelect: (sandbox: SandboxInfo) => void;
  selectedSandboxId?: string;
}

interface SandboxItemProps {
  sandbox: SandboxInfo;
  isSelected: boolean;
  onSelect: () => void;
  onDelete: () => void;
  onStart: () => void;
  onStop: () => void;
}
```

**Features**:
- Real-time updates via polling or WebSocket
- Search/filter functionality
- Group by sandbox type
- Status indicators with tooltips
- Context menu for actions

### 2. TaskLogs Component
```typescript
interface TaskLogsProps {
  sandboxId: string;
  taskId?: string;
  onTaskComplete: (exitCode: number) => void;
}

interface LogEntry {
  type: RunClaudeTaskResponseType;
  content: string;
  timestamp: number;
  exitCode?: number;
  isFinished?: boolean;
}
```

**Features**:
- Real-time streaming via Server-Sent Events
- Syntax highlighting for different log types
- Search within logs
- Export logs functionality
- Auto-scroll toggle
- Log level filtering

### 3. TaskPrompt Component
```typescript
interface TaskPromptProps {
  sandboxId: string;
  onTaskStart: (taskDescription: string) => void;
  isTaskRunning: boolean;
}
```

**Features**:
- Multi-line text input with syntax highlighting
- Character count and limit
- Task history (last 10 tasks)
- Quick action buttons for common tasks
- Validation for empty/too long descriptions

### 4. ModifiedFilesList Component
```typescript
interface ModifiedFilesListProps {
  sandboxId: string;
  onFileSelect?: (filePath: string) => void;
}

interface FileItem {
  path: string;
  status: 'added' | 'modified' | 'deleted';
  size?: number;
  lastModified?: string;
}
```

**Features**:
- Real-time file system monitoring
- File type icons
- Diff preview (future enhancement)
- File size and modification time
- Filter by file type or status

### 5. Terminal Component
```typescript
interface TerminalProps {
  sandboxId: string;
  onCommandExecute: (command: string) => void;
  isConnected: boolean;
}
```

**Features**:
- WebSocket connection for real-time output
- Command history with up/down arrow navigation
- Multiple terminal tabs
- Resizable height
- Copy/paste support
- Clear terminal functionality

## API Integration

### DispenseClient Usage
The application will use the provided `@api-client-ts` client for all API communications:

```typescript
// API service wrapper
class DashboardAPIService {
  private client: DispenseClient;

  constructor() {
    this.client = new DispenseClient({
      baseUrl: window.location.origin,
      timeout: 30000
    });
  }

  // Sandbox operations
  async getSandboxes(projectId: string): Promise<SandboxInfo[]> {
    const response = await this.client.listSandboxes({
      group: projectId,
      show_local: true,
      show_remote: true
    });
    return response.sandboxes;
  }

  // Task operations
  async runTask(sandboxId: string, description: string): Promise<void> {
    await this.client.runClaudeTask({
      sandbox_identifier: sandboxId,
      task_description: description
    }, (response) => {
      // Handle streaming response
      this.handleTaskResponse(response);
    });
  }

  // File operations
  async getModifiedFiles(sandboxId: string): Promise<FileItem[]> {
    // Implementation depends on available API endpoints
    // This might require extending the API client
  }
}
```

### Real-time Updates
- **Task Logs**: Server-Sent Events for streaming task output
- **Sandbox Status**: Polling every 5 seconds or WebSocket
- **File Changes**: Polling every 10 seconds or file system watcher
- **Terminal Output**: WebSocket connection

## State Management

### Global State Structure
```typescript
interface DashboardState {
  // Current selection
  selectedProject: string;
  selectedSandbox: SandboxInfo | null;
  selectedTask: string | null;

  // Data
  sandboxes: SandboxInfo[];
  tasks: TaskLog[];
  modifiedFiles: FileItem[];
  terminalOutput: TerminalLine[];

  // UI state
  isLoading: boolean;
  error: string | null;
  sidebarCollapsed: boolean;
  terminalHeight: number;
}
```

### Context Providers
- `SandboxContext`: Manages sandbox selection and operations
- `TaskContext`: Handles task execution and logging
- `FileContext`: Manages file system state
- `TerminalContext`: Controls terminal operations

## Styling and Theming

### Design System
- **Color Palette**: 
  - Primary: #2563eb (blue)
  - Success: #059669 (green)
  - Warning: #d97706 (orange)
  - Error: #dc2626 (red)
  - Neutral: #6b7280 (gray)
- **Typography**: System font stack with monospace for code
- **Spacing**: 4px base unit (4px, 8px, 12px, 16px, 24px, 32px)
- **Border Radius**: 4px for small elements, 8px for cards
- **Shadows**: Subtle shadows for depth

### Responsive Design
- **Desktop**: Full three-panel layout
- **Tablet**: Collapsible sidebar
- **Mobile**: Single panel with navigation tabs

## Build and Embedding Requirements

### Build Configuration
```json
// project.json
{
  "name": "dashboard",
  "type": "application",
  "targets": {
    "build": {
      "executor": "@nx/webpack:webpack",
      "options": {
        "outputPath": "dist/apps/dashboard",
        "index": "apps/dashboard/public/index.html",
        "main": "apps/dashboard/src/main.tsx",
        "tsConfig": "apps/dashboard/tsconfig.json",
        "assets": ["apps/dashboard/public"],
        "optimization": true,
        "extractLicenses": true,
        "sourceMap": false,
        "namedChunks": false
      }
    }
  }
}
```

### Go Embedding Integration
The built dashboard assets will be embedded into the Dispense binary:

```go
//go:embed dist/apps/dashboard/*
var dashboardAssets embed.FS

// Serve dashboard at /dashboard route
func (s *Server) setupDashboardRoutes() {
    dashboardFS, _ := fs.Sub(dashboardAssets, "dist/apps/dashboard")
    http.Handle("/dashboard/", http.StripPrefix("/dashboard/", http.FileServer(http.FS(dashboardFS))))
}
```

### Static Asset Handling
- All assets will be built with content hashes for caching
- API calls will be relative to the current origin
- No external dependencies (all bundled)

## Development Requirements

### Dependencies
```json
{
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "@api-client-ts": "workspace:*",
    "zustand": "^4.4.0",
    "date-fns": "^2.30.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.0",
    "@types/react-dom": "^18.2.0",
    "typescript": "^5.0.0",
    "@nx/webpack": "^21.5.2"
  }
}
```

### Development Scripts
- `nx serve dashboard`: Development server with hot reload
- `nx build dashboard`: Production build
- `nx test dashboard`: Unit tests
- `nx lint dashboard`: ESLint checking

### Testing Strategy
- **Unit Tests**: Component testing with React Testing Library
- **Integration Tests**: API service testing with MSW
- **E2E Tests**: Playwright for critical user flows

## Performance Requirements

### Bundle Size
- Initial bundle: < 500KB gzipped
- Lazy load non-critical components
- Tree-shake unused API client methods

### Runtime Performance
- Task log updates: < 100ms latency
- Sandbox list refresh: < 500ms
- File list updates: < 1s
- Terminal output: < 50ms latency

### Memory Usage
- < 50MB memory footprint
- Efficient cleanup of event listeners
- Debounced API calls

## Security Considerations

### API Security
- All API calls include authentication headers
- CORS configuration for embedded context
- Input validation and sanitization

### Content Security Policy
```html
<meta http-equiv="Content-Security-Policy" 
      content="default-src 'self'; 
               script-src 'self' 'unsafe-inline'; 
               style-src 'self' 'unsafe-inline'; 
               connect-src 'self' ws: wss:;">
```

## Future Enhancements

### Phase 2 Features
- File editor integration
- Real-time collaboration
- Custom themes
- Plugin system
- Advanced terminal features (tabs, split panes)

### Phase 3 Features
- Mobile app
- Offline support
- Advanced analytics
- Custom dashboard widgets

## Acceptance Criteria

### Functional Requirements
- [ ] Display list of sandboxes for selected project
- [ ] Show real-time task logs with streaming
- [ ] Allow task execution via prompt input
- [ ] Display modified files list
- [ ] Provide functional terminal interface
- [ ] Handle errors gracefully with user feedback

### Non-Functional Requirements
- [ ] Loads in < 3 seconds on first visit
- [ ] Responsive design works on desktop and tablet
- [ ] Real-time updates work reliably
- [ ] Memory usage stays under 50MB
- [ ] Builds successfully with Nx
- [ ] Embeds correctly in Go binary

### Integration Requirements
- [ ] Uses provided API client correctly
- [ ] Serves from `/dashboard` route in server mode
- [ ] Works with existing gRPC Gateway setup
- [ ] Handles authentication properly
