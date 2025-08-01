package main

import "time"

// BuildRequest represents a compilation request sent from client to server
type BuildRequest struct {
	ID           string            `json:"id"`
	Environment  string            `json:"environment"`   // Environment name for reference
	Command      string            `json:"command"`       // Complete build command
	ProjectDir   string            `json:"project_dir"`   // Project directory
	ExecutionDir string            `json:"execution_dir"` // Execution directory (relative to project_dir)
	OutputPaths  []string          `json:"output_paths"`  // Output file patterns
	EnvVars      map[string]string `json:"env_vars"`      // Environment variables
	Files        map[string]string `json:"files"`         // filename -> file content
	ProjectName  string            `json:"project_name"`  // unique project identifier
}

// BuildResponse represents the compilation result sent back from server
type BuildResponse struct {
	ID          string            `json:"id"`
	Success     bool              `json:"success"`
	Output      string            `json:"output"`
	Error       string            `json:"error,omitempty"`
	Duration    time.Duration     `json:"duration"`
	OutputFiles map[string]string `json:"output_files,omitempty"` // compiled files: filename -> base64 content
}

// ClientInfo represents client registration information
type ClientInfo struct {
	ID       string `json:"id"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Capacity int    `json:"capacity"`
}

// ServerInfo represents server registration information
type ServerInfo struct {
	ID       string `json:"id"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Capacity int    `json:"capacity"`
	Version  string `json:"version"`
}

// ServerStatusInfo represents server status for web interface
type ServerStatusInfo struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	Port      int    `json:"port"`
	Capacity  int    `json:"capacity"`
	Available bool   `json:"available"`
	Version   string `json:"version"`
}
