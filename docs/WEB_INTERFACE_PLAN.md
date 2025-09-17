# Web Interface Implementation Plan

## Overview

This document outlines the plan for adding a web interface to the pkmgradegap CLI tool. The approach maintains backward compatibility while providing a modern web-based user experience for Pokemon card price analysis.

## Architecture Overview

The current CLI tool has excellent architecture with clean provider interfaces, sophisticated analysis capabilities, and rich data structures. For the web interface, we recommend a **hybrid approach** that adds HTTP/WebSocket servers to the existing binary while reusing the core analysis engine.

## Recommended Architecture

### 1. Hybrid CLI/Web Server Binary
- Single binary that can run in CLI mode (current behavior) or web server mode
- Command: `./pkmgradegap --web --port 8080` to start web interface
- Preserves all existing CLI functionality

### 2. Service Layer Extraction
```
cmd/pkmgradegap/
├── main.go (CLI logic)
├── server.go (HTTP server)
└── handlers/ (HTTP handlers)

internal/
├── service/ (NEW: business logic services)
│   ├── analysis_service.go
│   ├── set_service.go
│   └── monitoring_service.go
├── api/ (NEW: JSON API handlers)
└── web/ (NEW: embedded static assets)
```

### 3. Technology Stack
- **Backend**: Go net/http with gorilla/mux for routing
- **Frontend**: Vue.js or React (lightweight SPA)
- **Real-time**: WebSockets for progress updates
- **Styling**: Tailwind CSS for clean, responsive UI
- **Build**: Embed static assets in Go binary

### 4. Key Web Features

#### Dashboard
- Set selection dropdown (replaces `--set` flag)
- Analysis type tabs (rank, alerts, trends, etc.)
- Configuration panels (costs, filters, toggles)
- Real-time progress bars during analysis

#### Results Display
- Interactive tables with sorting/filtering
- Charts for price trends and volatility
- Export options (CSV, JSON, PDF)
- eBay listings integration with clickable links

#### Monitoring Interface
- Snapshot management (view, compare, delete)
- Alert dashboard with severity levels
- Historical trend visualizations
- Market timing recommendations

#### Configuration Management
- Web forms replacing CLI flags
- API key management interface
- Cache and data file management
- Settings persistence

### 5. API Design

#### Core Endpoints
```
GET  /api/sets                    # List available sets
GET  /api/sets/{id}/cards         # Get cards in set
POST /api/analysis/rank           # Run rank analysis
POST /api/analysis/alerts         # Compare snapshots
GET  /api/snapshots               # List snapshots
POST /api/snapshots/compare       # Compare two snapshots
GET  /api/history                 # Get historical data
```

#### WebSocket Events
```
progress_update    # Real-time progress during data fetching
analysis_complete  # Analysis finished with results
error_occurred     # Error handling with user-friendly messages
```

### 6. Implementation Benefits

#### Leverages Existing Code
- Reuses all provider interfaces (PokeTCGIO, PriceCharting, eBay)
- Uses existing analysis algorithms and scoring
- Maintains caching and monitoring systems
- Preserves data structures and business logic

#### Enhanced User Experience
- Visual progress indicators instead of CLI spinners
- Interactive data exploration vs static CSV
- Real-time updates and error handling
- Persistent configuration and session state

#### Better Data Visualization
- Price trend charts using Chart.js or D3
- Volatility heatmaps
- ROI comparison graphs
- Population scarcity visualizations

## Implementation Plan

### Phase 1: Foundation (1-2 days)
1. **Add Web Server Flag**
   - Add `--web` and `--port` flags to main.go
   - Create basic HTTP server that serves static files
   - Add graceful shutdown handling

2. **Service Layer Extraction**
   - Move core analysis logic from main.go into service classes
   - Create `AnalysisService`, `SetService`, `MonitoringService`
   - Ensure services return structured data (not just CSV)

3. **Basic API Structure**
   - Set up gorilla/mux router with CORS support
   - Create base API handlers with JSON responses
   - Add error handling middleware

### Phase 2: Core APIs (2-3 days)
1. **Data APIs**
   - `/api/sets` - List available sets with metadata
   - `/api/sets/{id}/cards` - Get cards with pagination
   - `/api/config` - Configuration management

2. **Analysis APIs**
   - `/api/analysis/rank` - POST with config, returns scored results
   - `/api/analysis/raw-vs-psa10` - Simple price comparison
   - `/api/analysis/trends` - Historical analysis

3. **WebSocket Foundation**
   - WebSocket hub for real-time updates
   - Progress event structure
   - Client connection management

### Phase 3: Frontend Foundation (2-3 days)
1. **Basic SPA Setup**
   - HTML/CSS/JS or Vue.js setup
   - Responsive layout with navigation
   - API client with error handling

2. **Core UI Components**
   - Set selection dropdown
   - Analysis configuration forms
   - Results table with sorting
   - Loading states and progress bars

### Phase 4: Advanced Features (3-4 days)
1. **Monitoring Dashboard**
   - Snapshot management interface
   - Alert visualization
   - Historical trend charts

2. **Data Visualization**
   - Price trend charts (Chart.js)
   - ROI comparison graphs
   - Population scarcity heatmaps

3. **Export and Integration**
   - CSV/JSON download buttons
   - eBay integration display
   - Snapshot comparison tools

### Phase 5: Polish and Deployment (1-2 days)
1. **Asset Embedding**
   - Embed static assets in Go binary
   - Single binary deployment
   - Production build optimization

2. **Documentation and Testing**
   - Update CLAUDE.md with web interface instructions
   - Basic integration tests
   - User documentation

## File Structure Changes

### New Directories
```
internal/
├── service/           # Business logic services
├── api/              # HTTP API handlers
└── web/              # Static web assets

cmd/pkmgradegap/
└── handlers/         # HTTP route handlers
```

### Key New Files
- `cmd/pkmgradegap/server.go` - Web server setup
- `internal/service/analysis_service.go` - Analysis business logic
- `internal/service/set_service.go` - Set management
- `internal/service/monitoring_service.go` - Monitoring/alerts
- `internal/api/handlers.go` - API route handlers
- `internal/web/index.html` - Main frontend entry point

## Configuration Changes

### New CLI Flags
```bash
--web                 # Enable web server mode
--port 8080          # Web server port (default: 8080)
--web-assets ./web   # Custom web assets directory (development)
```

### Environment Variables
```bash
PKMGRADEGAP_WEB_PORT=8080      # Alternative to --port
PKMGRADEGAP_WEB_ASSETS=/path   # Alternative to --web-assets
```

## Usage Examples

### Start Web Interface
```bash
# Basic web server
./pkmgradegap --web

# Custom port
./pkmgradegap --web --port 3000

# Development mode with custom assets
./pkmgradegap --web --web-assets ./frontend/dist
```

### CLI Mode (unchanged)
```bash
# All existing CLI commands continue to work
./pkmgradegap --set "Surging Sparks" --analysis rank
./pkmgradegap --list-sets
```

## Technical Considerations

### Backward Compatibility
- All existing CLI functionality remains unchanged
- Web mode is opt-in via `--web` flag
- No breaking changes to existing APIs or data structures

### Performance
- Service layer enables better resource management
- WebSocket connections for real-time updates
- Caching strategy remains unchanged
- Concurrent request handling with Go routines

### Security
- CORS configuration for local-only access
- Input validation on all API endpoints
- No authentication needed (local-only tool)
- Rate limiting to prevent API abuse

### Data Management
- Existing cache and snapshot systems work unchanged
- Web interface can manage snapshots through UI
- Configuration persistence for web-specific settings
- File upload/download for snapshot management

## Next Steps

1. **Phase 1 Implementation**: Start with adding basic web server capability
2. **Service Extraction**: Move business logic out of main.go
3. **API Development**: Build core REST endpoints
4. **Frontend Development**: Create basic web interface
5. **Integration**: Connect frontend to backend APIs
6. **Testing**: Ensure both CLI and web modes work correctly
7. **Documentation**: Update user guides and technical docs

This plan provides a solid foundation for adding web capabilities while maintaining the tool's existing strengths and ensuring smooth migration for current users.