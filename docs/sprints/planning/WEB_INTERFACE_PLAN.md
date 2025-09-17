# Web Interface Implementation Plan (Simplified)

## Overview

This document outlines the plan for adding a web interface to the pkmgradegap CLI tool. Based on feedback, this is a **simplified, local-first approach** that maintains the tool's single-user, local nature while adding essential web capabilities without over-engineering.

## Architecture Overview (Simplified)

The current CLI tool has excellent architecture with clean provider interfaces, sophisticated analysis capabilities, and rich data structures. For the web interface, we use a **minimal hybrid approach** that adds basic HTTP/WebSocket servers to the existing binary while reusing the core analysis engine.

## Simplified Architecture

### 1. Hybrid CLI/Web Server Binary
- Single binary that can run in CLI mode (current behavior) or web server mode
- Command: `./pkmgradegap --web --port 8080` to start web interface
- Preserves all existing CLI functionality
- Binds to 127.0.0.1 by default (local-only)

### 2. Minimal File Structure
```
cmd/pkmgradegap/
├── main.go (existing CLI logic)
└── server.go (HTTP server + handlers)

internal/
└── web/
    └── index.html (single HTML page, no build step)
```

### 3. Simplified Technology Stack
- **Backend**: Go net/http (no gorilla/mux needed)
- **Frontend**: Single HTML page with vanilla JavaScript (no React/Vue/build step)
- **Real-time**: WebSockets for progress updates
- **Styling**: Minimal embedded CSS
- **Build**: Embed static assets with `//go:embed`

### 4. Key Web Features (Simplified)

#### Single Dashboard Page
- **Top**: Set selector + configuration inputs (analysis type, filters, costs)
- **Middle**: Progress bar with WebSocket updates
- **Bottom**: Results table with sorting + export buttons
- **Sidebar/Tab**: History/snapshots (simple list view)

#### Results Display
- Sortable HTML table (reusing existing CSV/JSON data)
- Client-side CSV/JSON export
- Simple diff view for snapshot comparisons
- No complex charts initially (can add Chart.js later)

#### Configuration
- Web forms replacing common CLI flags
- No authentication (local-only tool)
- No CORS complexity (assets served by same binary)

### 5. Simplified API Design

#### Minimal Endpoints
```
GET  /                           # Serves index.html
GET  /api/sets                   # List available sets
POST /api/analysis/run           # Run any analysis type
GET  /api/snapshots              # List snapshots (later)
POST /api/snapshots/compare      # Compare snapshots (later)
GET  /ws                         # WebSocket for progress
```

#### WebSocket Progress Format
```
{"stage":"fetch", "message":"PriceCharting: 37/120", "done":false}
{"stage":"analysis", "message":"Scoring cards...", "done":false}
{"stage":"complete", "message":"Analysis complete", "done":true}
{"stage":"error", "message":"Error details", "error":true}
```

### 6. Implementation Benefits

#### Leverages Existing Code
- Reuses all provider interfaces (PokeTCGIO, PriceCharting, eBay)
- Uses existing analysis algorithms and scoring
- Maintains caching and monitoring systems
- Preserves data structures and business logic
- **Same data output** as CLI (just formatted as JSON/HTML)

#### Enhanced User Experience
- Visual progress indicators instead of CLI spinners
- Sortable table vs static CSV
- Real-time updates and error handling
- No complex navigation or deep UI hierarchy

#### Simplified Maintenance
- Single HTML file, no build pipeline
- No framework dependencies or version conflicts
- Direct reuse of existing CSV/JSON export logic
- Minimal API surface area

## Simplified Implementation Plan

### Phase 1: MVP (1-2 days)
1. **Add Web Server Flag**
   - Add `--web` and `--port` flags to main.go
   - Create basic HTTP server with `//go:embed` for index.html
   - Bind to 127.0.0.1 by default (local-only)

2. **Single Analysis Endpoint**
   - POST `/api/analysis/run` - accepts config JSON, returns same data as CLI
   - GET `/api/sets` - list available sets
   - Reuse existing analysis functions directly

3. **WebSocket Progress**
   - GET `/ws` - WebSocket endpoint for progress updates
   - Connect existing progress hooks to WebSocket broadcaster
   - Simple progress message format

4. **Single HTML Page**
   - Form for set selection and configuration
   - Results table with client-side sorting
   - Progress area with WebSocket updates
   - Client-side CSV/JSON export

### Phase 2: Snapshots & History (1-2 days)
1. **Snapshot Management**
   - GET `/api/snapshots` - list saved snapshots
   - POST `/api/snapshots/compare` - simple diff between two snapshots
   - Add history tab to UI

2. **Export Enhancement**
   - Server-side CSV download route (optional)
   - Snapshot save/load through UI

### Phase 3: Polish (Optional)
1. **Data Visualization**
   - Add Chart.js for basic trend visualization
   - Simple volatility indicators

2. **Testing**
   - Ensure API returns same data as CLI for identical inputs
   - Basic integration tests

## Simplified File Structure Changes

### Minimal New Files
```
cmd/pkmgradegap/
└── server.go         # HTTP server + API handlers

internal/
└── web/
    └── index.html    # Single HTML page (no build step)
```

### Key New Files
- `cmd/pkmgradegap/server.go` - Web server setup + all API handlers
- `internal/web/index.html` - Complete single-page interface (provided below)

## Configuration Changes

### New CLI Flags
```bash
--web                 # Enable web server mode
--port 8080          # Web server port (default: 8080)
```

### Environment Variables
```bash
PKMGRADEGAP_WEB_PORT=8080      # Alternative to --port
```

## Usage Examples

### Start Web Interface
```bash
# Basic web server (binds to 127.0.0.1:8080)
./pkmgradegap --web

# Custom port
./pkmgradegap --web --port 3000
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
- Direct reuse of existing analysis functions
- WebSocket connections for real-time updates
- Caching strategy remains unchanged
- Single-user design simplifies concurrency

### Security & Simplicity
- No CORS needed (assets served by same binary)
- Input validation on API endpoints
- No authentication needed (local-only tool)
- Binds to 127.0.0.1 by default

### Data Management
- Existing cache and snapshot systems work unchanged
- Web interface displays same data as CLI
- Client-side export using existing data structures
- No complex state management needed

## Provided: Complete HTML Interface

The following complete `index.html` file implements the entire web interface without any build step or framework dependencies:

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>pkmgradegap — Local UI</title>
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    :root{color-scheme:light dark}
    body{margin:0;font:14px/1.4 system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,"Helvetica Neue",Arial}
    header{display:flex;gap:12px;align-items:center;padding:12px 16px;border-bottom:1px solid #ccc}
    main{padding:16px;max-width:1200px;margin:auto}
    .row{display:flex;flex-wrap:wrap;gap:12px}
    .card{border:1px solid #ccc;border-radius:10px;padding:12px}
    .grow{flex:1}
    label{display:block;font-size:12px;color:#666;margin-bottom:4px}
    input,select,button{padding:8px;border:1px solid #888;border-radius:8px;background:transparent;color:inherit}
    button.primary{background:#2563eb;color:white;border-color:#2563eb}
    button:disabled{opacity:.6}
    table{width:100%;border-collapse:collapse;margin-top:12px}
    th,td{border:1px solid #ddd;padding:8px;text-align:left}
    th{cursor:pointer;background:rgba(0,0,0,.04)}
    .muted{color:#666}
    .progress{font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;background:#f6f6f6;border:1px solid #ddd;border-radius:8px;padding:8px;max-height:160px;overflow:auto}
    .chip{display:inline-block;padding:2px 8px;border-radius:999px;border:1px solid #bbb;font-size:12px;margin-right:6px}
    .danger{color:#b00020}
    .ok{color:#0b7a0b}
    .right{margin-left:auto}
    .toolbar{display:flex;gap:8px;align-items:center}
  </style>
</head>
<body>
  <header>
    <strong>pkmgradegap</strong>
    <span class="muted">• local UI</span>
    <span class="right"></span>
    <div class="toolbar">
      <button id="btnExportCSV" title="Download CSV">Export CSV</button>
      <button id="btnExportJSON" title="Download JSON">Export JSON</button>
    </div>
  </header>

  <main>
    <section class="row">
      <div class="card grow">
        <h3 style="margin:0 0 8px">Run Analysis</h3>
        <div class="row">
          <div>
            <label for="setSelect">Set</label>
            <select id="setSelect"></select>
          </div>
          <div>
            <label for="analysisType">Analysis</label>
            <select id="analysisType">
              <option value="rank">Rank (recommended)</option>
              <option value="raw-vs-psa10">Raw vs PSA10</option>
              <option value="psa9-cgc95-bgs95-vs-psa10">Multi-grade vs PSA10</option>
            </select>
          </div>
          <div>
            <label for="maxAgeYears">Max Age (years)</label>
            <input id="maxAgeYears" type="number" min="0" step="1" value="10" />
          </div>
          <div>
            <label for="minRawUSD">Min Raw USD</label>
            <input id="minRawUSD" type="number" min="0" step="0.01" value="0.50" />
          </div>
          <div>
            <label for="minDeltaUSD">Min Delta USD</label>
            <input id="minDeltaUSD" type="number" min="0" step="1" value="25" />
          </div>
          <div>
            <label for="jpnWeight">Japanese Weight ×</label>
            <input id="jpnWeight" type="number" min="1" step="0.05" value="1.00" />
          </div>
          <div>
            <label for="withEbay">eBay Listings</label>
            <select id="withEbay">
              <option value="false">Off (faster)</option>
              <option value="true">On</option>
            </select>
          </div>
          <div>
            <label for="topN">Top N</label>
            <input id="topN" type="number" min="1" step="1" value="25" />
          </div>
        </div>

        <div class="row" style="margin-top:12px">
          <button class="primary" id="btnRun">Run</button>
          <span id="runStatus" class="muted"></span>
          <span class="right chip" id="paramChip"></span>
        </div>
      </div>

      <div class="card" style="min-width:320px;max-width:420px">
        <h3 style="margin:0 0 8px">Progress</h3>
        <div class="progress" id="progressLog" aria-live="polite"></div>
      </div>
    </section>

    <section style="margin-top:16px">
      <div class="card">
        <div class="row" style="align-items:center;margin-bottom:8px">
          <h3 style="margin:0">Results</h3>
          <span class="right muted" id="resultMeta"></span>
        </div>
        <div id="tableMount"></div>
      </div>
    </section>
  </main>

<script>
  // ---- State ----
  let currentResults = { columns: [], rows: [] };
  let ws;

  // ---- Utils ----
  function $(id){ return document.getElementById(id) }
  function esc(s){ return String(s ?? '').replace(/[&<>"']/g, m => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m])) }
  function asMoney(v){ if (v === null || v === undefined || v === '') return ''; const n=Number(v); return isFinite(n)? '$'+n.toFixed(2):String(v) }
  function csvEscapeCell(s){
    s = String(s ?? '');
    const riskyStart = ['=','+','-','@','\t'];
    if (s && riskyStart.includes(s[0])) s = "'" + s;
    if (s.includes('"') || s.includes(',') || s.includes('\n')) {
      return '"' + s.replace(/"/g,'""') + '"';
    }
    return s;
  }

  function renderTable(data){
    const {columns, rows} = data;
    if (!columns || !columns.length){
      $('tableMount').innerHTML = '<div class="muted">No results yet.</div>'; return;
    }
    // Build table
    let thead = '<thead><tr>' + columns.map(c=>`<th data-key="${esc(c)}">${esc(c)}</th>`).join('') + '</tr></thead>';
    let tbody = '<tbody>' + rows.map(r=>{
      return '<tr>' + columns.map(c=>{
        let v = r[c];
        if (typeof v === 'number' && /usd|delta|score|price|cost/i.test(c)) v = asMoney(v);
        return `<td>${esc(v)}</td>`;
      }).join('') + '</tr>';
    }).join('') + '</tbody>';
    $('tableMount').innerHTML = `<table id="resultTable">${thead}${tbody}</table>`;

    // Sort on header click (simple client-side ascending/descending)
    const table = $('resultTable');
    const ths = table.querySelectorAll('th');
    ths.forEach((th, idx)=>{
      let asc = true;
      th.addEventListener('click', ()=>{
        const key = th.dataset.key;
        const sorted = [...rows].sort((a,b)=>{
          const av = a[key], bv = b[key];
          const an = Number(av), bn = Number(bv);
          const bothNum = isFinite(an) && isFinite(bn);
          if (bothNum) return asc ? (an - bn) : (bn - an);
          return asc ? String(av).localeCompare(String(bv)) : String(bv).localeCompare(String(av));
        });
        currentResults.rows = sorted;
        renderTable(currentResults);
        asc = !asc;
      });
    });
  }

  function setBusy(b){
    $('btnRun').disabled = b;
    $('runStatus').textContent = b ? 'Running…' : '';
  }

  function logProgress(msg, cls){
    const el = $('progressLog');
    const line = document.createElement('div');
    if (cls) line.className = cls;
    line.textContent = msg;
    el.appendChild(line);
    el.scrollTop = el.scrollHeight;
  }

  function connectWS(){
    try {
      if (ws) ws.close();
    } catch {}
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${proto}//${location.host}/ws`);
    ws.onopen = ()=> logProgress('Connected to progress stream.');
    ws.onmessage = (e)=>{
      try{
        const m = JSON.parse(e.data);
        const {stage, message, done, error} = m;
        logProgress(`[${stage}] ${message}`, error ? 'danger' : (done ? 'ok' : ''));
        if (error) setBusy(false);
      }catch{
        logProgress(e.data);
      }
    };
    ws.onerror = ()=> logProgress('Progress stream error', 'danger');
    ws.onclose = ()=> logProgress('Progress stream closed.');
  }

  async function loadSets(){
    const res = await fetch('/api/sets');
    if (!res.ok){ $('setSelect').innerHTML = '<option>(failed)</option>'; return; }
    const sets = await res.json();
    $('setSelect').innerHTML = sets.map(s=>`<option value="${esc(s.id)}">${esc(s.name)} (${esc(s.releaseDate||'')})</option>`).join('');
  }

  async function runAnalysis(){
    const payload = {
      setId: $('setSelect').value,
      analysis: $('analysisType').value,
      maxAgeYears: Number($('maxAgeYears').value),
      minRawUSD: Number($('minRawUSD').value),
      minDeltaUSD: Number($('minDeltaUSD').value),
      japaneseWeight: Number($('jpnWeight').value),
      withEbay: $('withEbay').value === 'true',
      top: Number($('topN').value)
    };
    $('paramChip').textContent = `age≤${payload.maxAgeYears}, raw≥$${payload.minRawUSD.toFixed(2)}, Δ≥$${payload.minDeltaUSD.toFixed(0)}, JP×${payload.japaneseWeight}`;
    setBusy(true);
    $('progressLog').innerHTML = '';
    connectWS();
    try{
      const res = await fetch('/api/analysis/run', {
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body: JSON.stringify(payload)
      });
      if (!res.ok){
        const txt = await res.text();
        logProgress(`Error: ${txt}`, 'danger');
        setBusy(false);
        return;
      }
      const data = await res.json();
      currentResults = data;
      $('resultMeta').textContent = `${data.rows.length} rows • ${new Date().toLocaleString()}`;
      renderTable(data);
    }catch(err){
      logProgress('Network error: '+err, 'danger');
    }finally{
      setBusy(false);
    }
  }

  function exportCSV(){
    if (!currentResults.columns?.length) return;
    const {columns, rows} = currentResults;
    const header = columns.map(csvEscapeCell).join(',');
    const body = rows.map(r=>columns.map(c=>csvEscapeCell(r[c])).join(',')).join('\n');
    const csv = header + '\n' + body;
    const blob = new Blob([csv], {type:'text/csv'});
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'pkmgradegap_results.csv';
    a.click();
    setTimeout(()=>URL.revokeObjectURL(a.href), 1000);
  }

  function exportJSON(){
    const blob = new Blob([JSON.stringify(currentResults,null,2)], {type:'application/json'});
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'pkmgradegap_results.json';
    a.click();
    setTimeout(()=>URL.revokeObjectURL(a.href), 1000);
  }

  // ---- Wire up ----
  $('btnRun').addEventListener('click', runAnalysis);
  $('btnExportCSV').addEventListener('click', exportCSV);
  $('btnExportJSON').addEventListener('click', exportJSON);

  // init
  loadSets().then(()=>connectWS());
</script>
</body>
</html>
```

## Server Implementation Notes

For `cmd/pkmgradegap/server.go`, implement:

1. **Serve embedded HTML**: Use `//go:embed` to serve the index.html at `/`
2. **GET `/api/sets`**: Return `[]struct{id, name, releaseDate}` from existing set provider
3. **POST `/api/analysis/run`**: Accept JSON payload, run existing analysis, return:
   ```json
   {
     "columns": ["Card", "No", "RawUSD", "PSA10", "Delta", "Score"],
     "rows": [{"Card": "...", "No": "...", "RawUSD": 12.34, ...}],
     "params": {}
   }
   ```
4. **GET `/ws`**: WebSocket endpoint that broadcasts progress from existing pipeline
5. **Bind to 127.0.0.1**: Default local-only binding for security

## Next Steps

1. **Phase 1**: Implement server.go with the three endpoints above
2. **Phase 2**: Add snapshot management if needed
3. **Testing**: Ensure API returns same data as CLI for identical inputs

This simplified plan cuts development time significantly while providing all essential web interface functionality.