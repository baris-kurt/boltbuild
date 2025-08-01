package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Client manages build requests and server connections
type Client struct {
	servers           map[string]*ServerConnection
	serversMux        sync.RWMutex
	pendingBuilds     map[string]chan *BuildResponse
	pendingMux        sync.RWMutex
	discoveredServers map[string]ServerInfo
	discoveryMux      sync.RWMutex
}

// ServerConnection represents a connection to a build server
type ServerConnection struct {
	info ServerInfo
	conn net.Conn
	busy bool
	mux  sync.Mutex
}

// NewClient creates a new client instance
func NewClient() *Client {
	return &Client{
		servers:           make(map[string]*ServerConnection),
		pendingBuilds:     make(map[string]chan *BuildResponse),
		discoveredServers: make(map[string]ServerInfo),
	}
}

// Start begins server discovery and connection management
func (c *Client) Start() error {
	LogInfo("Client started, discovering build servers...")

	// Start server discovery
	go c.discoverServers()

	// Start connection manager
	go c.manageConnections()

	// Keep running
	select {}
}

// discoverServers discovers available build servers on the network
func (c *Client) discoverServers() {
	for {
		// Try configured ports on local network
		c.scanForServers()
		time.Sleep(globalConfig.Client.Discovery.ScanInterval)
	}
}

// scanForServers scans for build servers on configured ports
func (c *Client) scanForServers() {
	ports := globalConfig.Client.Discovery.Ports

	// Determine network range
	var networkPrefix string
	var startIP, endIP int

	if globalConfig.Client.Discovery.NetworkRange.Auto {
		localIP := c.getLocalIP()
		networkPrefix = c.getNetworkPrefix(localIP)
		startIP = 1
		endIP = 254
	} else {
		networkPrefix = globalConfig.Client.Discovery.NetworkRange.Subnet
		startIP = globalConfig.Client.Discovery.NetworkRange.StartIP
		endIP = globalConfig.Client.Discovery.NetworkRange.EndIP
	}

	for i := startIP; i <= endIP; i++ {
		ip := fmt.Sprintf("%s.%d", networkPrefix, i)
		for _, port := range ports {
			go c.tryConnectToServer(ip, port)
		}
	}
}

// tryConnectToServer attempts to connect to a potential server
func (c *Client) tryConnectToServer(ip string, port int) {
	addr := fmt.Sprintf("%s:%d", ip, port)

	// Skip if already connected
	c.serversMux.RLock()
	_, exists := c.servers[addr]
	c.serversMux.RUnlock()
	if exists {
		return
	}

	// Try to connect with configured timeout
	conn, err := net.DialTimeout("tcp", addr, globalConfig.Client.Discovery.ConnectTimeout)
	if err != nil {
		return
	}

	// Try to read server info
	decoder := json.NewDecoder(conn)
	var serverInfo ServerInfo
	if err := decoder.Decode(&serverInfo); err != nil {
		conn.Close()
		return
	}

	// Verify this is a build server
	if !strings.HasPrefix(serverInfo.ID, "server-") {
		conn.Close()
		return
	}

	// Check version compatibility
	if serverInfo.Version != Version {
		LogDebugf("WARNING: Version mismatch with server %s! Client: %s, Server: %s", serverInfo.ID, Version, serverInfo.Version)
	}

	LogInfof("Discovered build server %s at %s (capacity: %d, version: %s)", serverInfo.ID, addr, serverInfo.Capacity, serverInfo.Version)

	// Add to discovered servers
	c.discoveryMux.Lock()
	c.discoveredServers[addr] = serverInfo
	c.discoveryMux.Unlock()

	// Start managing this connection
	go c.handleServerConnection(conn, serverInfo, addr)
}

// handleServerConnection manages a single server connection
func (c *Client) handleServerConnection(conn net.Conn, serverInfo ServerInfo, addr string) {
	defer conn.Close()

	serverConn := &ServerConnection{
		info: serverInfo,
		conn: conn,
		busy: false,
	}

	c.serversMux.Lock()
	c.servers[addr] = serverConn
	c.serversMux.Unlock()

	LogInfof("Connected to build server %s at %s (capacity: %d)", serverInfo.ID, addr, serverInfo.Capacity)

	// Keep connection alive and handle responses
	decoder := json.NewDecoder(conn)
	for {
		var response BuildResponse
		if err := decoder.Decode(&response); err != nil {
			LogInfof("Server %s disconnected: %v", serverInfo.ID, err)
			break
		}

		LogDebugf("Build %s completed by server %s: success=%v, output_files=%d", response.ID, serverInfo.ID, response.Success, len(response.OutputFiles))

		// Send response to waiting SubmitBuild call
		c.pendingMux.Lock()
		if responseChan, exists := c.pendingBuilds[response.ID]; exists {
			responseChan <- &response
			delete(c.pendingBuilds, response.ID)
		}
		c.pendingMux.Unlock()

		serverConn.mux.Lock()
		serverConn.busy = false
		serverConn.mux.Unlock()
	}

	// Remove server on disconnect
	c.serversMux.Lock()
	delete(c.servers, addr)
	c.serversMux.Unlock()

	// Remove from discovered servers
	c.discoveryMux.Lock()
	delete(c.discoveredServers, addr)
	c.discoveryMux.Unlock()
}

// manageConnections manages server connections and reconnections
func (c *Client) manageConnections() {
	for {
		time.Sleep(globalConfig.Client.Timeouts.HealthCheck)

		// Check for disconnected servers and try to reconnect
		c.discoveryMux.RLock()
		for addr, serverInfo := range c.discoveredServers {
			c.serversMux.RLock()
			_, connected := c.servers[addr]
			c.serversMux.RUnlock()

			if !connected {
				go c.reconnectToServer(addr, serverInfo)
			}
		}
		c.discoveryMux.RUnlock()
	}
}

// reconnectToServer attempts to reconnect to a disconnected server
func (c *Client) reconnectToServer(addr string, serverInfo ServerInfo) {
	conn, err := net.DialTimeout("tcp", addr, globalConfig.Client.Timeouts.Reconnect)
	if err != nil {
		return
	}

	// Try to read server info again
	decoder := json.NewDecoder(conn)
	var newServerInfo ServerInfo
	if err := decoder.Decode(&newServerInfo); err != nil {
		conn.Close()
		return
	}

	// Verify it's the same server
	if newServerInfo.ID != serverInfo.ID {
		conn.Close()
		return
	}

	LogInfof("Reconnected to build server %s at %s", serverInfo.ID, addr)
	go c.handleServerConnection(conn, newServerInfo, addr)
}

// SubmitBuild submits a build request to an available server with file transfer
func (c *Client) SubmitBuild(environment, entry, projectDir string, args []string) (*BuildResponse, error) {
	// Generate unique build ID and project name
	buildID := generateID()
	projectName := fmt.Sprintf("project_%s", buildID)

	// Read all files from the project directory
	files, err := c.readProjectFiles(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read project files: %v", err)
	}

	// Get environment configuration
	env, exists := globalConfig.GetBuildEnvironment(environment)
	if !exists {
		return nil, fmt.Errorf("environment %s not found in client configuration", environment)
	}

	request := BuildRequest{
		ID:           buildID,
		Environment:  environment,
		Command:      env.Command,
		ProjectDir:   env.ProjectDir,
		ExecutionDir: env.ExecutionDir,
		OutputPaths:  env.OutputPaths,
		EnvVars:      env.EnvVars,
		Files:        files,
		ProjectName:  projectName,
	}

	// Find available server
	server := c.findAvailableServer()
	if server == nil {
		return nil, fmt.Errorf("no available servers")
	}

	// Check version compatibility before submitting build
	if server.info.Version != Version {
		return nil, fmt.Errorf("version mismatch: client version %s, server %s version %s. Please ensure all components are using the same version", Version, server.info.ID, server.info.Version)
	}

	// Create response channel for this build
	responseChan := make(chan *BuildResponse, 1)
	c.pendingMux.Lock()
	c.pendingBuilds[buildID] = responseChan
	c.pendingMux.Unlock()

	// Mark server as busy
	server.mux.Lock()
	server.busy = true
	server.mux.Unlock()

	// Send build request with files
	encoder := json.NewEncoder(server.conn)
	if err := encoder.Encode(request); err != nil {
		server.mux.Lock()
		server.busy = false
		server.mux.Unlock()

		// Clean up pending build
		c.pendingMux.Lock()
		delete(c.pendingBuilds, buildID)
		c.pendingMux.Unlock()

		return nil, fmt.Errorf("failed to send build request: %v", err)
	}

	LogDebugf("Build %s submitted to server %s with %d files", buildID, server.info.ID, len(files))

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		// Save compiled files to output directory if build was successful
		if response.Success && len(response.OutputFiles) > 0 {
			if err := c.saveOutputFiles(projectDir, response.OutputFiles); err != nil {
				LogDebugf("Warning: Failed to save output files: %v", err)
			}
		}

		// Execute post-build script if build was successful and script is configured
		if response.Success && env.PostBuildScript != "" {
			if err := c.executePostBuildScript(env.PostBuildScript, projectDir, env); err != nil {
				LogDebugf("Warning: Failed to execute post-build script: %v", err)
				// Note: We don't fail the build for post-build script errors
			}
		}

		return response, nil
	case <-time.After(globalConfig.Client.Timeouts.Build):
		// Cleanup on timeout
		c.pendingMux.Lock()
		delete(c.pendingBuilds, buildID)
		c.pendingMux.Unlock()

		return nil, fmt.Errorf("build timeout after %v", globalConfig.Client.Timeouts.Build)
	}
}

// SubmitBuildToServer submits a build request to a specific server
func (c *Client) SubmitBuildToServer(environment, entry, projectDir, workdir string, args []string, serverAddr string) (*BuildResponse, error) {
	// Generate unique build ID and project name
	buildID := generateID()
	projectName := fmt.Sprintf("project_%s", buildID)

	// Get environment configuration
	env, exists := globalConfig.GetBuildEnvironment(environment)
	if !exists {
		return nil, fmt.Errorf("environment %s not found in client configuration", environment)
	}

	// Read all files from the project directory
	files, err := c.readProjectFiles(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read project files: %v", err)
	}

	request := BuildRequest{
		ID:           buildID,
		Environment:  environment,
		Command:      env.Command,
		ProjectDir:   env.ProjectDir,
		ExecutionDir: env.ExecutionDir,
		OutputPaths:  env.OutputPaths,
		EnvVars:      env.EnvVars,
		Files:        files,
		ProjectName:  projectName,
	}

	// Find the specific server
	server := c.findServerByAddress(serverAddr)
	if server == nil {
		return nil, fmt.Errorf("server %s not found or not connected", serverAddr)
	}

	// Check version compatibility before submitting build
	if server.info.Version != Version {
		return nil, fmt.Errorf("version mismatch: client version %s, server %s version %s. Please ensure all components are using the same version", Version, server.info.ID, server.info.Version)
	}

	// Check if server is available
	server.mux.Lock()
	if server.busy {
		server.mux.Unlock()
		return nil, fmt.Errorf("server %s is currently busy", serverAddr)
	}
	server.busy = true
	server.mux.Unlock()

	// Create response channel for this build
	responseChan := make(chan *BuildResponse, 1)
	c.pendingMux.Lock()
	c.pendingBuilds[buildID] = responseChan
	c.pendingMux.Unlock()

	// Send build request with files
	encoder := json.NewEncoder(server.conn)
	if err := encoder.Encode(request); err != nil {
		server.mux.Lock()
		server.busy = false
		server.mux.Unlock()

		// Clean up pending build
		c.pendingMux.Lock()
		delete(c.pendingBuilds, buildID)
		c.pendingMux.Unlock()

		return nil, fmt.Errorf("failed to send build request to %s: %v", serverAddr, err)
	}

	LogDebugf("Build %s submitted to server %s (%s) with %d files", buildID, server.info.ID, serverAddr, len(files))

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		// Save compiled files to output directory if build was successful
		if response.Success && len(response.OutputFiles) > 0 {
			if err := c.saveOutputFiles(workdir, response.OutputFiles); err != nil {
				LogDebugf("Warning: Failed to save output files: %v", err)
			}
		}

		// Execute post-build script if build was successful and script is configured
		if response.Success && env.PostBuildScript != "" {
			if err := c.executePostBuildScript(env.PostBuildScript, workdir, env); err != nil {
				LogDebugf("Warning: Failed to execute post-build script: %v", err)
				// Note: We don't fail the build for post-build script errors
			}
		}

		return response, nil
	case <-time.After(globalConfig.Client.Timeouts.Build):
		// Cleanup on timeout
		c.pendingMux.Lock()
		delete(c.pendingBuilds, buildID)
		c.pendingMux.Unlock()

		return nil, fmt.Errorf("build timeout after %v", globalConfig.Client.Timeouts.Build)
	}
}

// findServerByAddress finds a server by its address
func (c *Client) findServerByAddress(serverAddr string) *ServerConnection {
	c.serversMux.RLock()
	defer c.serversMux.RUnlock()

	for _, server := range c.servers {
		currentAddr := fmt.Sprintf("%s:%d", server.info.Address, server.info.Port)
		if currentAddr == serverAddr {
			return server
		}
	}
	return nil
}

// findAvailableServer returns an available server or nil
func (c *Client) findAvailableServer() *ServerConnection {
	c.serversMux.RLock()
	defer c.serversMux.RUnlock()

	for _, server := range c.servers {
		server.mux.Lock()
		busy := server.busy
		server.mux.Unlock()

		if !busy {
			return server
		}
	}
	return nil
}

// GetServerStatus returns the status of all connected servers
func (c *Client) GetServerStatus() map[string]ServerStatusInfo {
	c.serversMux.RLock()
	defer c.serversMux.RUnlock()

	status := make(map[string]ServerStatusInfo)
	for id, server := range c.servers {
		server.mux.Lock()
		status[id] = ServerStatusInfo{
			ID:        server.info.ID,
			Address:   server.info.Address,
			Port:      server.info.Port,
			Capacity:  server.info.Capacity,
			Available: !server.busy,
			Version:   server.info.Version,
		}
		server.mux.Unlock()
	}
	return status
}

// readProjectFiles reads all files from the project directory
func (c *Client) readProjectFiles(workdir string) (map[string]string, error) {
	files := make(map[string]string)

	err := filepath.WalkDir(workdir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Get file info for size check
		info, err := d.Info()
		if err != nil {
			return err
		}

		// Skip binary files and large files (>1MB)
		if info.Size() > 1024*1024 {
			return nil
		}

		// Skip certain file extensions
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".exe" || ext == ".dll" || ext == ".so" || ext == ".dylib" || ext == ".o" || ext == ".obj" {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", path, err)
		}

		// Get relative path from workdir
		relPath, err := filepath.Rel(workdir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %v", path, err)
		}

		// Normalize path to use forward slashes for cross-platform compatibility
		normalizedRelPath := filepath.ToSlash(relPath)

		// Store file content with normalized relative path as key
		files[normalizedRelPath] = string(content)
		return nil
	})

	if err != nil {
		return nil, err
	}

	LogDebugf("Read %d files from project directory: %s", len(files), workdir)
	return files, nil
}

// saveOutputFiles saves compiled output files to the work directory
func (c *Client) saveOutputFiles(workdir string, outputFiles map[string]string) error {
	for relPath, encodedContent := range outputFiles {
		// Decode base64 content
		content, err := base64.StdEncoding.DecodeString(encodedContent)
		if err != nil {
			LogDebugf("Warning: Failed to decode file %s: %v", relPath, err)
			continue
		}

		// Normalize path separators for the current OS
		// The server always sends paths with forward slashes, so convert to native separators
		normalizedRelPath := filepath.FromSlash(relPath)
		
		// Create full output path directly in workdir
		outputPath := filepath.Join(workdir, normalizedRelPath)

		// Create directory if needed
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			LogDebugf("Warning: Failed to create directory %s: %v", dir, err)
			continue
		}

		// Write file
		if err := os.WriteFile(outputPath, content, 0755); err != nil {
			LogDebugf("Warning: Failed to write file %s: %v", outputPath, err)
			continue
		}

		LogDebugf("Saved output file: %s", outputPath)
	}

	LogDebugf("Saved %d output files to project directory %s", len(outputFiles), workdir)
	return nil
}

// generateID creates a random ID for build requests
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// getLocalIP returns the local IP address of the client
func (c *Client) getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "192.168.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// getNetworkPrefix returns the network prefix (e.g., "192.168.1" from "192.168.1.100")
func (c *Client) getNetworkPrefix(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], ".")
	}
	return "192.168.1"
}

// executePostBuildScript executes the configured post-build script after a successful build
func (c *Client) executePostBuildScript(scriptPath, projectDir string, env *BuildEnvironment) error {
	// Import os/exec at the top of the file if not already imported
	var cmd *exec.Cmd

	// Check if the script path is absolute or relative
	var fullScriptPath string
	if filepath.IsAbs(scriptPath) {
		fullScriptPath = scriptPath
	} else {
		// If relative, make it relative to the project directory
		fullScriptPath = filepath.Join(projectDir, scriptPath)
	}

	// Check if the script/executable exists
	if _, err := os.Stat(fullScriptPath); os.IsNotExist(err) {
		return fmt.Errorf("post-build script not found: %s", fullScriptPath)
	}

	// Determine how to execute the script based on its extension
	ext := strings.ToLower(filepath.Ext(fullScriptPath))
	switch ext {
	case ".bat", ".cmd":
		// Windows batch file
		cmd = exec.Command("cmd", "/C", fullScriptPath)
	case ".sh":
		// Shell script
		cmd = exec.Command("bash", fullScriptPath)
	case ".ps1":
		// PowerShell script
		cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", fullScriptPath)
	case ".py":
		// Python script
		cmd = exec.Command("python", fullScriptPath)
	case ".exe", "":
		// Executable or file without extension (assume executable)
		cmd = exec.Command(fullScriptPath)
	default:
		// Try to execute directly
		cmd = exec.Command(fullScriptPath)
	}

	// Set working directory to project directory
	cmd.Dir = projectDir

	// Set environment variables from build environment configuration
	cmd.Env = os.Environ()
	for key, value := range env.EnvVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Add some useful environment variables for the script
	cmd.Env = append(cmd.Env, fmt.Sprintf("BOLTBUILD_PROJECT_DIR=%s", projectDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("BOLTBUILD_ENVIRONMENT=%s", env.Name))
	cmd.Env = append(cmd.Env, fmt.Sprintf("BOLTBUILD_OUTPUT_DIR=%s", filepath.Join(projectDir, "output")))

	LogDebugf("Executing post-build script: %s", fullScriptPath)

	// Execute the script and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("post-build script failed: %v\nOutput: %s", err, string(output))
	}

	LogDebugf("Post-build script completed successfully. Output: %s", string(output))
	return nil
}
