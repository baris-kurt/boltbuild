# BoltBuild

**BoltBuild** is a remote build system that allows you to offload compilation tasks to multiple build servers across your network. It automatically discovers available build servers and distributes compilation jobs to speed up your development workflow.

## Features

- **Remote Building**: Automatically distribute compilation tasks across multiple build servers.
- **Auto-Discovery**: Automatically find and connect to build servers on your network.
- **Web Interface**:  Dashboard to monitor servers and submit builds.
- **Multi Environment Support**: Configure build environments dynamically.
- **Real-time Monitoring**: Track build progress and server status in real-time.
- **Flexible Configuration**: YAML-based configuration.
- **Cross-Platform**: Works on Windows, Linux, and macOS.

## Architecture

BoltBuild consists of two main components:

- **Server Mode**: Build workers that execute compilation tasks.
- **Client Mode**: Build coordinator with web interface that discovers servers and distributes work.

```
                    BoltBuild Architecture
                    
    Developer Machine                    Build Server Network
    ┌─────────────────┐                 ┌─────────────────────┐
    │                 │                 │                     │
    │  ┌───────────┐  │    Discovery    │  ┌───────────────┐  │
    │  │    Web    │  │ <─────────────> │  │ Build Server  │  │
    │  │ Interface │  │                 │  │   (Port 8080) │  │
    │  └───────────┘  │                 │  └───────────────┘  │
    │        │        │                 │                     │
    │        │        │                 │  ┌───────────────┐  │
    │  ┌───────────┐  │   Build Jobs    │  │ Build Server  │  │
    │  │  Client   │  │ ──────────────> │  │   (Port 8081) │  │
    │  │Coordinator│  │                 │  └───────────────┘  │
    │  └───────────┘  │                 │                     │
    │        │        │                 │  ┌───────────────┐  │
    │  ┌───────────┐  │   Results       │  │ Build Server  │  │
    │  │   File    │  │ <────────────── │  │   (Port 8082) │  │
    │  │  Manager  │  │                 │  └───────────────┘  │
    │  └───────────┘  │                 │                     │
    └─────────────────┘                 └─────────────────────┘
    
    Flow:
    1. Client scans network for build servers.
    2. Developer submits code via web interface.
    3. Client distributes build jobs to available servers.
    4. Servers compile code and return results.
    5. Client collects outputs and presents to developer.
```

## Quick Start

### 1. Download and Build

```bash
git clone <repository-url>
cd boltbuild
go build -o boltbuild
```

### 2. Start a Build Server

On machines you want to use as build workers:

```bash
./boltbuild server
```

The server will start on port 8080 by default and wait for client connections.

### 3. Start the Client

On your development machine:

```bash
./boltbuild client
```

The client will:
- Automatically discover build servers on your network
- Start a web interface on http://localhost:8081
- Begin coordinating build requests

### 4. Access the Web Interface

Open your browser and navigate to:
```
http://localhost:8081
```

You'll see a dashboard showing:
- Connected build servers
- Available build environments
- Build submission interface

## Configuration

BoltBuild uses YAML configuration files. On first run, it creates a default `config.yaml` file.

See `config-example.yaml` for a comprehensive configuration example with:
- Custom network ranges
- Multiple build environments
- Environment variables
- Post-build scripts
- Extended timeout settings


## Security Considerations

- BoltBuild is designed for trusted networks (development environments)
- No authentication is implemented - use on secure networks only
- Build servers execute arbitrary code - only connect trusted clients
- Consider firewall rules to restrict access to build ports

## Development


### Project Structure

```
├── main.go      # Application entry point
├── server.go    # Build server implementation  
├── client.go    # Build client implementation
├── web.go       # Web interface
├── config.go    # Configuration management
├── types.go     # Data structures
└── logging.go   # Logging utilities
```

## Requirements

- **Go 1.21+** for building from source
- **Network connectivity** between client and servers
- **Build configuration(s)** configured on the client, instructions on how to build

## License

This project is licensed under the GNUv3 License - see the LICENSE file for details.