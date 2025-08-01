package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// WebServer provides HTTP interface for the client
type WebServer struct {
	client *Client
	port   int
}

// NewWebServer creates a new web server instance
func NewWebServer(client *Client, port int) *WebServer {
	return &WebServer{
		client: client,
		port:   port,
	}
}

// Start begins the web server
func (ws *WebServer) Start() error {
	r := mux.NewRouter()

	// Static routes
	r.HandleFunc("/", ws.handleHome).Methods("GET")
	r.HandleFunc("/api/servers", ws.handleServersAPI).Methods("GET")
	r.HandleFunc("/api/environments", ws.handleEnvironmentsAPI).Methods("GET")
	r.HandleFunc("/api/build", ws.handleBuildAPI).Methods("POST")
	r.HandleFunc("/api/version", ws.handleVersionAPI).Methods("GET")

	LogInfof("Web server starting on port %d", ws.port)
	return http.ListenAndServe(":"+strconv.Itoa(ws.port), r)
}

// handleHome serves the main dashboard
func (ws *WebServer) handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Montserrat:ital,wght@0,100..900;1,100..900&display=swap" rel="stylesheet">
    <title>BoltBuild - Client Dashboard</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #031C26;
            color: #A4FFF0;
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 1400px;
            margin: 0 auto;
        }
        
        .header {
            text-align: center;
            padding: 30px 0;
            margin-bottom: 40px;
        }
        
        .header .logo {
            width: 200px;
            height: auto;
            margin-bottom: 20px;
            filter: drop-shadow(0 4px 8px rgba(164, 255, 240, 0.3));
        }
        
        .header h1 {
            font-family: "Montserrat", Inter;
            letter-spacing: -3px;
            color: #A4FFF0;
            font-size: 3rem;
            font-weight: 700;
            text-shadow: 0 0 6px #a4fff09e;
        }
        
        .header h1 span {
            color: #fff;
            font-size: 3rem;
            font-weight: 300;
            margin-bottom: 10px;
            text-shadow: 0 2px 4px rgba(164, 255, 240, 0.3);
        }

        
        .header p {
            color: rgba(164, 255, 240, 0.8);
            font-weight: 300;
        }
        
        .dashboard-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 30px;
            margin-bottom: 30px;
        }
        
        @media (max-width: 768px) {
            .dashboard-grid {
                grid-template-columns: 1fr;
            }
        }
        
        .card {
            background: rgba(164, 255, 240, 0.05);
            backdrop-filter: blur(10px);
            padding: 30px;
            border-radius: 20px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.3);
            border: 1px solid rgba(164, 255, 240, 0.2);
        }
        
        .card h2 {
            color: #A4FFF0;
            font-size: 1.5rem;
            font-weight: 600;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
            gap: 10px;
        }
        
        .servers-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
            gap: 20px;
        }
        
        .server-card {
            background: rgba(164, 255, 240, 0.08);
            padding: 20px;
            border-radius: 15px;
            box-shadow: 0 8px 25px rgba(0,0,0,0.2);
            border: 2px solid rgba(164, 255, 240, 0.2);
            transition: all 0.3s ease;
            position: relative;
            overflow: hidden;
            cursor: pointer;
        }
        
        .server-card.selected {
            border-color: #A4FFF0;
            box-shadow: 0 8px 20px rgba(164, 255, 240, 0.2);
            transform: translateY(-3px);
        }
        
        .server-card.selected::before {
            background: linear-gradient(90deg, #A4FFF0 0%, #7BFFF0 100%);
        }
        
        .server-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 4px;
            background: linear-gradient(90deg, #A4FFF0 0%, #7BFFF0 100%);
        }
        
        .server-available {
            border-color: rgba(164, 255, 240, 0.3);
            background: rgba(164, 255, 240, 0.05);
        }
        
        .server-available::before {
            background: linear-gradient(90deg, rgba(164, 255, 240, 0.6) 0%, rgba(123, 255, 240, 0.6) 100%);
        }
        
        .server-busy {
            border-color: #f56565;
        }
        
        .server-busy::before {
            background: linear-gradient(90deg, #f56565 0%, #e53e3e 100%);
        }
        
        .server-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 15px 35px rgba(164, 255, 240, 0.2);
        }
        
        .server-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
        }
        
        .server-id {
            font-weight: 600;
            color: #A4FFF0;
            font-size: 1.1rem;
        }
        
        .server-status {
            padding: 6px 12px;
            border-radius: 20px;
            font-size: 0.85rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        
        .status-available {
            background: rgba(164, 255, 240, 0.1);
            color: rgba(164, 255, 240, 0.9);
        }
        
        .status-busy {
            background: #fed7d7;
            color: #742a2a;
        }
        
        .version-mismatch {
            border: 2px solid #ff6b6b !important;
            background: rgba(255, 107, 107, 0.05) !important;
        }

        .version-mismatch::before {
            background: #ff6b6b !important;
        }
        
        .version-mismatch:hover {
            border-color: #ff6b6b !important;
            background: rgba(255, 107, 107, 0.1) !important;
        }
        
        .server-info {
            color: rgba(164, 255, 240, 0.7);
            font-size: 0.9rem;
            line-height: 1.5;
        }
        
        .form-group {
            margin-bottom: 20px;
        }
        
        .form-group label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #A4FFF0;
        }
        
        .form-control {
            width: 100%;
            padding: 12px 16px;
            border: 2px solid rgba(164, 255, 240, 0.3);
            border-radius: 10px;
            font-size: 1rem;
            transition: all 0.3s ease;
            background: rgba(164, 255, 240, 0.05);
            color: #A4FFF0;
        }
        
        .form-control:focus {
            outline: none;
            border-color: #A4FFF0;
            box-shadow: 0 0 0 3px rgba(164, 255, 240, 0.2);
        }
        
        .form-control option {
            background: #031C26;
            color: #A4FFF0;
        }
        
        .btn {
            background: linear-gradient(135deg, #A4FFF0 0%, #7BFFF0 100%);
            color: #031C26;
            padding: 14px 28px;
            border: none;
            border-radius: 10px;
            font-size: 1rem;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s ease;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 25px rgba(164, 255, 240, 0.4);
        }
        
        .btn:active {
            transform: translateY(0);
        }
        
        .result {
            margin-top: 25px;
            padding: 20px;
            border-radius: 10px;
            font-size: 0.95rem;
            line-height: 1.6;
        }
        
        .result-success {
            background: rgba(164, 255, 240, 0.1);
            border: 2px solid #A4FFF0;
            color: #A4FFF0;
        }
        
        .result-error {
            background: rgba(245, 101, 101, 0.1);
            border: 2px solid #f56565;
            color: #A4FFF0;
        }
        
        .loading {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 3px solid rgba(164, 255, 240, 0.3);
            border-top: 3px solid #A4FFF0;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
        
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
            gap: 15px;
            margin-bottom: 20px;
        }
        
        .stat-item {
            text-align: center;
            padding: 15px;
            background: rgba(164, 255, 240, 0.08);
            border-radius: 10px;
            border: 1px solid rgba(164, 255, 240, 0.2);
        }
        
        .stat-number {
            font-size: 2rem;
            font-weight: 700;
            color: #A4FFF0;
        }
        
        .stat-label {
            font-size: 0.85rem;
            color: rgba(164, 255, 240, 0.7);
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        
        /* Modal styles */
        .modal {
            display: none;
            position: fixed;
            z-index: 1000;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.8);
            backdrop-filter: blur(5px);
        }
        
        .modal-content {
            background: #031C26;
            margin: 5% auto;
            padding: 0;
            border-radius: 20px;
            width: 90%;
            max-width: 1000px;
            max-height: 80vh;
            box-shadow: 0 25px 50px rgba(0, 0, 0, 0.5);
            border: 2px solid rgba(164, 255, 240, 0.3);
            overflow: hidden;
        }
        
        .modal-header {
            background: linear-gradient(135deg, rgba(164, 255, 240, 0.1) 0%, rgba(123, 255, 240, 0.1) 100%);
            padding: 20px 30px;
            border-bottom: 1px solid rgba(164, 255, 240, 0.2);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        
        .modal-title {
            color: #A4FFF0;
            font-size: 1.5rem;
            font-weight: 600;
            margin: 0;
        }
        
        .close {
            background: none;
            border: none;
            color: #A4FFF0;
            font-size: 2rem;
            font-weight: bold;
            cursor: pointer;
            padding: 0;
            width: 30px;
            height: 30px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: all 0.3s ease;
        }
        
        .close:hover {
            background: rgba(164, 255, 240, 0.1);
            transform: scale(1.1);
        }
        
        .modal-body {
            padding: 30px;
            max-height: 60vh;
            overflow-y: auto;
        }
        
        .output-content {
            background: rgba(164, 255, 240, 0.05);
            padding: 20px;
            border-radius: 10px;
            border: 1px solid rgba(164, 255, 240, 0.2);
            font-family: 'Courier New', monospace;
            font-size: 0.9rem;
            line-height: 1.4;
            color: #A4FFF0;
            white-space: pre-wrap;
            word-break: break-word;
            max-height: 50vh;
            overflow-y: auto;
        }
        
        .output-content::-webkit-scrollbar {
            width: 8px;
        }
        
        .output-content::-webkit-scrollbar-track {
            background: rgba(164, 255, 240, 0.1);
            border-radius: 4px;
        }
        
        .output-content::-webkit-scrollbar-thumb {
            background: rgba(164, 255, 240, 0.3);
            border-radius: 4px;
        }
        
        .output-content::-webkit-scrollbar-thumb:hover {
            background: rgba(164, 255, 240, 0.5);
        }
        
        .btn-view-output {
            background: linear-gradient(135deg, rgba(164, 255, 240, 0.2) 0%, rgba(123, 255, 240, 0.2) 100%);
            color: #A4FFF0;
            padding: 8px 16px;
            border: 1px solid rgba(164, 255, 240, 0.3);
            border-radius: 8px;
            font-size: 0.9rem;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.3s ease;
            margin-top: 10px;
            display: inline-block;
            text-decoration: none;
        }
        
        .btn-view-output:hover {
            background: linear-gradient(135deg, rgba(164, 255, 240, 0.3) 0%, rgba(123, 255, 240, 0.3) 100%);
            border-color: #A4FFF0;
            transform: translateY(-1px);
        }

    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>bolt<span>build</span></h1>
            <p>Remote Build System</p>
            <p style="margin-top: 5px; font-size: 0.9rem; color: rgba(164, 255, 240, 0.6);">Client Version: <span id="client-version">Loading...</span></p>
        </div>
        
        <div class="dashboard-grid">
            <div class="card">
                <h2>📊 Build Servers Status</h2>
                <div class="stats" id="server-stats">
                    <div class="stat-item">
                        <div class="stat-number" id="total-servers">0</div>
                        <div class="stat-label">Total Servers</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-number" id="available-servers">0</div>
                        <div class="stat-label">Available</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-number" id="busy-servers">0</div>
                        <div class="stat-label">Busy</div>
                    </div>
                </div>
                <div id="servers-container" class="servers-grid">
                    <div style="text-align: center; padding: 40px; color: #718096;">
                        <div class="loading"></div>
                        <p style="margin-top: 15px;">Loading servers...</p>
                    </div>
                </div>
            </div>
            
            <div class="card">
                <h2>🔨 Submit Build Request</h2>
                <form id="build-form">
                    <div class="form-group">
                        <label for="selected-server">Selected Server:</label>
                        <div id="selected-server" class="form-control" style="color: rgba(164, 255, 240, 0.7); font-style: italic;">No server selected - Click on a server to select</div>
                    </div>
                    <div class="form-group">
                        <label for="environment">Build Environment:</label>
                        <select id="environment" name="environment" class="form-control" required>
                            <option value="">Loading environments...</option>
                        </select>
                    </div>
                    <button type="submit" class="btn">🚀 Start Build</button>
                </form>
                <div id="build-result"></div>
            </div>
        </div>
    </div>
    
    <!-- Modal for viewing build output -->
    <div id="outputModal" class="modal">
        <div class="modal-content">
            <div class="modal-header">
                <h2 class="modal-title" id="modalTitle">Build Output</h2>
                <button class="close" onclick="closeOutputModal()">&times;</button>
            </div>
            <div class="modal-body">
                <div id="modalOutput" class="output-content"></div>
            </div>
        </div>
    </div>
    
    <script>
        let selectedServer = null;
        
        // Modal functions
        function showOutputModal(title, output) {
            document.getElementById('modalTitle').textContent = title;
            document.getElementById('modalOutput').textContent = output;
            document.getElementById('outputModal').style.display = 'block';
            document.body.style.overflow = 'hidden'; // Prevent background scrolling
        }
        
        function closeOutputModal() {
            document.getElementById('outputModal').style.display = 'none';
            document.body.style.overflow = 'auto'; // Restore scrolling
        }
        
        // Close modal when clicking outside of it
        window.onclick = function(event) {
            const modal = document.getElementById('outputModal');
            if (event.target === modal) {
                closeOutputModal();
            }
        }
        
        // Close modal with Escape key
        document.addEventListener('keydown', function(event) {
            if (event.key === 'Escape') {
                closeOutputModal();
            }
        });
        
        function selectServer(serverAddr, serverInfo) {
            selectedServer = { addr: serverAddr, info: serverInfo };
            
            // Update UI
            document.querySelectorAll('.server-card').forEach(card => {
                card.classList.remove('selected');
            });
            
            const selectedCard = document.querySelector('[data-server-addr="' + serverAddr + '"]');
            if (selectedCard) {
                selectedCard.classList.add('selected');
            }
            
            // Update selected server display
            const selectedServerDiv = document.getElementById('selected-server');
            selectedServerDiv.innerHTML = '<strong>' + serverInfo.id + '</strong> - ' + serverInfo.address + ':' + serverInfo.port + ' (Capacity: ' + serverInfo.capacity + ')';
            selectedServerDiv.style.background = 'rgba(164, 255, 240, 0.1)';
            selectedServerDiv.style.color = '#A4FFF0';
            selectedServerDiv.style.fontStyle = 'normal';
        }
        
        function loadEnvironments() {
            fetch('/api/environments')
                .then(response => response.json())
                .then(data => {
                    const environmentSelect = document.getElementById('environment');
                    environmentSelect.innerHTML = '<option value="">Select build environment...</option>';
                    
                    Object.values(data).forEach(env => {
                        const option = document.createElement('option');
                        option.value = env.name;
                        option.textContent = env.name;
                        if (env.description) {
                            option.textContent += ' - ' + env.description;
                        }
                        environmentSelect.appendChild(option);
                    });
                })
                .catch(error => {
                    console.error('Error loading environments:', error);
                    const environmentSelect = document.getElementById('environment');
                    environmentSelect.innerHTML = '<option value="">Error loading environments</option>';
                });
        }
        
        function loadServers() {
            // Fetch both servers and client version for comparison
            Promise.all([
                fetch('/api/servers').then(response => response.json()),
                fetch('/api/version').then(response => response.json())
            ])
                .then(([serverData, versionData]) => {
                    const container = document.getElementById('servers-container');
                    const servers = Object.values(serverData);
                    const clientVersion = versionData.version;
                    
                    // Update stats
                    const totalServers = servers.length;
                    const availableServers = servers.filter(s => s.available).length;
                    const busyServers = totalServers - availableServers;
                    
                    document.getElementById('total-servers').textContent = totalServers;
                    document.getElementById('available-servers').textContent = availableServers;
                    document.getElementById('busy-servers').textContent = busyServers;
                    
                    if (totalServers === 0) {
                        container.innerHTML = '<div style="text-align: center; padding: 40px; color: rgba(164, 255, 240, 0.7); grid-column: 1 / -1;"><h3>No Build Servers Connected</h3><p>Start some build servers to begin compilation</p></div>';
                        return;
                    }
                    
                    container.innerHTML = '';
                    servers.forEach((server, index) => {
                        const serverAddr = server.address + ':' + server.port;
                        const versionMismatch = server.version !== clientVersion;
                        const serverCard = document.createElement('div');
                        
                        // Add version-mismatch class if versions don't match
                        let cardClasses = 'server-card ' + (server.available ? 'server-available' : 'server-busy');
                        if (versionMismatch) {
                            cardClasses += ' version-mismatch';
                        }
                        serverCard.className = cardClasses;
                        serverCard.setAttribute('data-server-addr', serverAddr);
                        
                        // Check if this server is currently selected
                        if (selectedServer && selectedServer.addr === serverAddr) {
                            serverCard.classList.add('selected');
                        }
                        
                        // Create version display with warning if mismatch
                        let versionDisplay = '<div><strong>Version:</strong> ' + server.version;
                        let clickHint = '<div style="margin-top: 10px; font-size: 0.8rem; color: #A4FFF0;">💡 Click to select this server</div>';
                        
                        if (versionMismatch) {
                            versionDisplay += ' <span style="color: #ff6b6b; font-weight: bold;">⚠️ MISMATCH</span>';
                            clickHint = '<div style="margin-top: 10px; font-size: 0.8rem; color: #ff6b6b;">⚠️ Version mismatch - builds will fail!</div>';
                        }
                        versionDisplay += '</div>';
                        
                        serverCard.innerHTML = '<div class="server-header">' +
                            '<div class="server-id">' + server.id + '</div>' +
                            '<div class="server-status ' + (server.available ? 'status-available' : 'status-busy') + '">' +
                                (server.available ? '✅ Available' : '⚡ Busy') +
                            '</div>' +
                        '</div>' +
                        '<div class="server-info">' +
                            '<div><strong>Address:</strong> ' + server.address + ':' + server.port + '</div>' +
                            '<div><strong>Capacity:</strong> ' + server.capacity + ' concurrent builds</div>' +
                            versionDisplay +
                            clickHint +
                        '</div>';
                        
                        // Add click event to select server
                        serverCard.addEventListener('click', () => {
                            selectServer(serverAddr, server);
                        });
                        
                        container.appendChild(serverCard);
                    });
                })
                .catch(error => {
                    console.error('Error loading servers:', error);
                    document.getElementById('servers-container').innerHTML = '<div style="text-align: center; padding: 40px; color: #f56565; grid-column: 1 / -1;"><h3>❌ Error Loading Servers</h3><p>Please check your connection</p></div>';
                });
        }
        
        document.getElementById('build-form').addEventListener('submit', function(e) {
            e.preventDefault();
            
            // Check if a server is selected
            if (!selectedServer) {
                alert('Please select a server first by clicking on one of the server cards above.');
                return;
            }
            
            const formData = new FormData(e.target);
            const buildRequest = {
                environment: formData.get('environment'),
                selectedServer: selectedServer.addr
            };
            
            const resultDiv = document.getElementById('build-result');
            resultDiv.innerHTML = '<div style="text-align: center; padding: 20px;"><div class="loading"></div><p style="margin-top: 15px; color: #A4FFF0; font-weight: 600;">Building project...</p></div>';
            
            fetch('/api/build', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(buildRequest)
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    let outputFilesInfo = '';
                    if (data.output_files && Object.keys(data.output_files).length > 0) {
                        outputFilesInfo = '<br><br><strong>📁 Output Files:</strong><br>';
                        for (const [filename, _] of Object.entries(data.output_files)) {
                            outputFilesInfo += '• ' + filename + '<br>';
                        }
                        outputFilesInfo += '<em>💾 Files saved to output/ directory</em>';
                    }
                    
                    // Store output for modal
                    window.lastBuildOutput = data.output;
                    window.lastBuildId = data.id;
                    
                    resultDiv.innerHTML = '<div class="result result-success">' +
                        '<h3>✅ Build Successful!</h3>' +
                        '<p><strong>Build ID:</strong> ' + data.id + '</p>' +
                        '<p><strong>Duration:</strong> ' + formatDuration(data.duration) + '</p>' +
                        '<button class="btn-view-output" onclick="showOutputModal(\'✅ Build Output - ' + data.id + '\', window.lastBuildOutput)">📋 View Build Output</button>' +
                        outputFilesInfo +
                    '</div>';
                } else {
                    // Store output for modal (including error output)
                    window.lastBuildOutput = data.output || 'No output available';
                    window.lastBuildId = data.id || 'Unknown';
                    
                    let viewOutputButton = '';
                    if (data.output) {
                        viewOutputButton = '<button class="btn-view-output" onclick="showOutputModal(\'❌ Build Error Output - ' + window.lastBuildId + '\', window.lastBuildOutput)">📋 View Error Output</button>';
                    }
                    
                    resultDiv.innerHTML = '<div class="result result-error">' +
                        '<h3>❌ Build Failed!</h3>' +
                        '<p><strong>Error:</strong> ' + (data.error || 'Unknown error') + '</p>' +
                        viewOutputButton +
                    '</div>';
                }
                loadServers();
            })
            .catch(error => {
                console.error('Error submitting build:', error);
                resultDiv.innerHTML = '<div class="result result-error">' +
                    '<h3>❌ Network Error!</h3>' +
                    '<p>Failed to submit build request. Please check your connection.</p>' +
                '</div>';
            });
        });
        
        // Function to format duration from nanoseconds to human readable format
          function formatDuration(nanoseconds) {
              const totalMilliseconds = Math.floor(nanoseconds / 1000000);
              const totalSeconds = Math.floor(nanoseconds / 1000000000);
              const hours = Math.floor(totalSeconds / 3600);
              const minutes = Math.floor((totalSeconds % 3600) / 60);
              const seconds = totalSeconds % 60;
              
              if (totalSeconds < 1) {
                  return totalMilliseconds + 'ms';
              } else if (hours > 0) {
                  return hours + 'h ' + minutes + 'm ' + seconds + 's';
              } else if (minutes > 0) {
                  return minutes + 'm ' + seconds + 's';
              } else {
                  return seconds + 's';
              }
          }
        
        function loadClientVersion() {
            fetch('/api/version')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('client-version').textContent = data.version;
                })
                .catch(error => {
                    console.error('Error loading client version:', error);
                    document.getElementById('client-version').textContent = 'Unknown';
                });
        }
        
        // Load environments and servers on page load
        loadClientVersion();
        loadEnvironments();
        loadServers();
        setInterval(loadServers, 3000);
    </script>
</body>
</html>`))
}

// handleServersAPI returns server status as JSON
func (ws *WebServer) handleServersAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := ws.client.GetServerStatus()

	data, err := json.Marshal(status)
	if err != nil {
		http.Error(w, "Failed to encode server status", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// handleVersionAPI returns client version as JSON
func (ws *WebServer) handleVersionAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	version := map[string]string{
		"version": Version,
	}

	data, err := json.Marshal(version)
	if err != nil {
		http.Error(w, "Failed to encode version", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// handleEnvironmentsAPI returns available build environments from config
func (ws *WebServer) handleEnvironmentsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get all build environments from config
	envs := make(map[string]interface{})
	for name, env := range globalConfig.Build.Environments {
		envs[name] = map[string]interface{}{
			"name":     name,
			"language": env.Name,
			"command":  env.Command,
		}
	}

	data, err := json.Marshal(envs)
	if err != nil {
		http.Error(w, "Failed to encode environments", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// handleBuildAPI handles build submission requests
func (ws *WebServer) handleBuildAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Environment    string `json:"environment"`
		SelectedServer string `json:"selectedServer"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get environment configuration to determine project directory for file reading
	env, exists := globalConfig.GetBuildEnvironment(req.Environment)
	if !exists {
		http.Error(w, fmt.Sprintf("Unknown environment: %s", req.Environment), http.StatusBadRequest)
		return
	}

	// Submit build request - client will handle environment configuration
	response, err := ws.client.SubmitBuildToServer(req.Environment, "", env.ProjectDir, env.ProjectDir, []string{}, req.SelectedServer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode build response", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}
