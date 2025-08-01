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

// Server represents a build server that accepts client connections
type Server struct {
	id         string
	port       int
	capacity   int
	clients    map[string]*ClientConnection
	clientsMux sync.RWMutex
}

// ClientConnection represents a connection from a client
type ClientConnection struct {
	conn net.Conn
	addr string
}

// NewServer creates a new server instance
func NewServer(port int, capacity int) *Server {
	id := generateServerID()
	return &Server{
		id:       id,
		port:     port,
		capacity: capacity,
		clients:  make(map[string]*ClientConnection),
	}
}

// Start begins listening for client connections
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}
	defer listener.Close()

	LogInfof("Build server %s started on port %d, waiting for clients...", s.id, s.port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			LogDebugf("Failed to accept connection: %v", err)
			continue
		}

		go s.handleClientConnection(conn)
	}
}

// handleClientConnection manages a single client connection
func (s *Server) handleClientConnection(conn net.Conn) {
	defer conn.Close()
	clientAddr := conn.RemoteAddr().String()

	// Register client
	clientConn := &ClientConnection{
		conn: conn,
		addr: clientAddr,
	}

	s.clientsMux.Lock()
	s.clients[clientAddr] = clientConn
	s.clientsMux.Unlock()

	LogInfof("Client connected from %s", clientAddr)

	// Send server info to client
	serverInfo := ServerInfo{
		ID:       s.id,
		Address:  s.getLocalIP(),
		Port:     s.port,
		Capacity: s.capacity,
		Version:  Version,
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(serverInfo); err != nil {
		LogDebugf("Failed to send server info to %s: %v", clientAddr, err)
		return
	}

	// Process build requests from this client
	decoder := json.NewDecoder(conn)
	for {
		var request BuildRequest
		if err := decoder.Decode(&request); err != nil {
			LogInfof("Client %s disconnected: %v", clientAddr, err)
			break
		}

		LogDebugf("Received build request %s for %s from %s", request.ID, request.Environment, clientAddr)
		response := s.processBuildRequest(request)

		if err := encoder.Encode(response); err != nil {
			LogDebugf("Failed to send response to %s: %v", clientAddr, err)
			break
		}
	}

	// Remove client on disconnect
	s.clientsMux.Lock()
	delete(s.clients, clientAddr)
	s.clientsMux.Unlock()
}

// processBuildRequest executes a build request and returns the result
func (s *Server) processBuildRequest(request BuildRequest) BuildResponse {
	start := time.Now()

	response := BuildResponse{
		ID: request.ID,
	}

	// Create temporary project directory
	projectDir, err := s.createProjectDirectory(request)
	if err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("Failed to create project directory: %v", err)
		response.Duration = time.Since(start)
		return response
	}

	// Clean up temporary directory based on configuration
	defer func() {
		if globalConfig.Build.TempDeletion {
			os.RemoveAll(projectDir)
		} else {
			LogDebugf("Temporary directory preserved: %s", projectDir)
		}
	}()

	// Write files to project directory
	if err := s.writeProjectFiles(projectDir, request.Files); err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("Failed to write project files: %v", err)
		response.Duration = time.Since(start)
		return response
	}

	// Execute build command based on language
	cmd, err := s.buildCommand(request, projectDir)
	if err != nil {
		response.Success = false
		response.Error = err.Error()
		response.Duration = time.Since(start)
		return response
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	response.Output = string(output)
	response.Duration = time.Since(start)

	if err != nil {
		response.Success = false
		response.Error = err.Error()
	} else {
		response.Success = true
		// Collect compiled output files
		outputFiles, err := s.collectOutputFiles(projectDir, request)
		if err != nil {
			LogDebugf("Warning: Failed to collect output files: %v", err)
		} else {
			response.OutputFiles = outputFiles
		}
	}

	LogDebugf("Build %s completed in %v, success: %v (files: %d, output: %d)", request.ID, response.Duration, response.Success, len(request.Files), len(response.OutputFiles))
	return response
}

// buildCommand creates the appropriate build command based on request configuration
func (s *Server) buildCommand(request BuildRequest, projectDir string) (*exec.Cmd, error) {
	// Parse the command string from the request
	cmdParts := strings.Fields(request.Command)
	if len(cmdParts) == 0 {
		return nil, fmt.Errorf("empty command in build request")
	}

	compiler := cmdParts[0]
	args := cmdParts[1:]

	// Determine execution directory
	executionDir := request.ExecutionDir
	if executionDir == "" {
		executionDir = projectDir // Fallback to project directory
	} else if !filepath.IsAbs(executionDir) {
		// If relative path, make it relative to project directory
		executionDir = filepath.Join(projectDir, executionDir)
	}

	// Create execution directory if it doesn't exist
	if err := os.MkdirAll(executionDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create execution directory: %v", err)
	}

	// Command will be executed in the execution directory
	LogDebugf("%s build command: %s %v (execution dir: %s)", request.Environment, compiler, args, executionDir)

	// Create command
	cmd := exec.Command(compiler, args...)
	cmd.Dir = executionDir

	// Set environment variables from request
	if len(request.EnvVars) > 0 {
		cmd.Env = os.Environ()
		for key, value := range request.EnvVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return cmd, nil
}

// createProjectDirectory creates a temporary directory for the build
func (s *Server) createProjectDirectory(request BuildRequest) (string, error) {
	// Create a temporary directory for project files
	tempDir := globalConfig.GetTempDir()
	projectDir := filepath.Join(tempDir, request.ProjectName)

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", err
	}

	return projectDir, nil
}

// writeProjectFiles writes all project files to the temporary directory
func (s *Server) writeProjectFiles(projectDir string, files map[string]string) error {
	for relativePath, content := range files {
		// Normalize path separators for the current OS
		normalizedRelPath := filepath.FromSlash(relativePath)
		fullPath := filepath.Join(projectDir, normalizedRelPath)

		// Create directory if it doesn't exist
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// collectOutputFiles collects compiled output files and returns them as base64
func (s *Server) collectOutputFiles(projectDir string, request BuildRequest) (map[string]string, error) {
	outputFiles := make(map[string]string)

	files, err := s.findFiles(projectDir)
	if err != nil {
		LogDebugf("Error finding files in project directory %s: %v", projectDir, err)
		return nil, err
	}

	LogDebugf("Found %d files in project directory %s for environment %s", len(files), projectDir, request.Environment)

	for _, file := range files {
		relativePath, err := filepath.Rel(projectDir, file)
		if err != nil {
			LogDebugf("Warning: Failed to get relative path for %s: %v", file, err)
			continue
		}
		// Normalize to use forward slashes and prefix with ./
		normalizedPath := "./" + filepath.ToSlash(relativePath)

		info, err := os.Stat(file)
		if err != nil {
			LogDebugf("Warning: Failed to stat file %s: %v", file, err)
			continue
		}

		LogDebugf("Checking file: %s (size: %d)", normalizedPath, info.Size())

		if s.isOutputFileNormalized(normalizedPath, request.OutputPaths) {
			content, err := os.ReadFile(file)
			if err != nil {
				LogDebugf("Warning: Failed to read output file %s: %v", file, err)
				continue
			}

			outputFiles[normalizedPath] = base64.StdEncoding.EncodeToString(content)
			LogDebugf("Added output file: %s (size: %d bytes)", normalizedPath, len(content))
		} else {
			LogDebugf("Skipped file (not output): %s", normalizedPath)
		}
	}

	LogDebugf("Collected %d output files for build %s", len(outputFiles), request.ID)
	return outputFiles, nil
}

// findFiles recursively finds all files in a directory
func (s *Server) findFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// isOutputFileNormalized matches output patterns against the normalized relative path (./...)
func (s *Server) isOutputFileNormalized(normalizedPath string, outputPaths []string) bool {
	if len(outputPaths) == 0 {
		return true
	}
	for _, pattern := range outputPaths {
		// Always use forward slashes for pattern and path
		patternNorm := filepath.ToSlash(pattern)
		matched, err := filepath.Match(patternNorm, normalizedPath)
		if err == nil && matched {
			return true
		}
		// Also check basename for patterns like "main.*"
		matched, err = filepath.Match(patternNorm, filepath.Base(normalizedPath))
		if err == nil && matched {
			return true
		}
	}
	return false
}

// generateServerID generates a unique server ID using computer name
func generateServerID() string {
	hostname, err := os.Hostname()
	if err != nil {
		// Fallback to random ID if hostname is not available
		bytes := make([]byte, 8)
		rand.Read(bytes)
		return fmt.Sprintf("server-%s", hex.EncodeToString(bytes))
	}
	return fmt.Sprintf("server-%s", hostname)
}

// getLocalIP returns the local IP address of the server
func (s *Server) getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
