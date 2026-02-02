# xk6-sftp Development Guide

This document explains the architecture, design decisions, and testing approach for the xk6-sftp extension.

## Architecture Overview

### k6 Extension Model

k6 extensions must handle **concurrent Virtual Users (VUs)**. Each VU runs in its own goroutine, executing your test script simultaneously. This creates a challenge: how do you manage stateful resources like SFTP connections without race conditions?

```
┌─────────────────────────────────────────────────────────┐
│                      k6 Runtime                         │
├─────────────────────────────────────────────────────────┤
│VU 1 (goroutine)    VU 2 (goroutine)    VU 3 (goroutine) │
│       │                   │                   │         │
│       ▼                   ▼                   ▼         │
│  ┌─────────┐         ┌─────────┐         ┌─────────┐    │
│  │ Client  │         │ Client  │         │ Client  │    │
│  │(per-VU) │         │(per-VU) │         │(per-VU) │    │
│  └────┬────┘         └────┬────┘         └────┬────┘    │
│       │                   │                   │         │
│       ▼                   ▼                   ▼         │
│  ┌──────────┐        ┌──────────┐        ┌──────────┐   │
│  │Connection│        │Connection│        │Connection│   │
│  │(owns SSH)│        │(owns SSH)│        │(owns SSH)│   │
│  └──────────┘        └──────────┘        └──────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Three-Layer Design

The extension uses three layers to achieve thread safety:

#### 1. Module (Singleton, Stateless)

```go
type Module struct{}
```

Registered once at startup via `init()`. Contains no fields—it's purely a factory. Since it has no mutable state, there's nothing to race on.

#### 2. Client (Per-VU)

```go
type Client struct{}
```

k6 creates one `Client` per VU by calling `NewModuleInstance()`. Each VU gets its own instance, providing isolation. If you need access to VU-specific context (iteration number, VU ID, metrics), add a `vu modules.VU` field.

#### 3. Connection (Per-Call, Caller-Owned)

```go
type Connection struct {
    sshClient  *ssh.Client
    sftpClient *sftp.Client
}
```

Created by `Connect()` and returned to JavaScript. The caller owns it and is responsible for calling `Close()`. Each VU manages its own connections independently.

### Why This Pattern?

**The Problem:** A naive implementation stores the SSH client in the module struct:

```go
// WRONG: Shared state causes races
type Sftp struct {
    Client *ssh.Client  // All VUs read/write this!
}
```

When VU 1 calls `connect(serverA)` and VU 2 calls `connect(serverB)`, they overwrite each other's connection.

**The Solution:** Return connections instead of storing them:

```go
// RIGHT: Each VU gets its own Connection
func (c *Client) Connect(...) (*Connection, error) {
    // Create new connection
    return &Connection{...}, nil  // Caller owns this
}
```

Now each VU has its own variable holding its own `Connection`. No shared mutable state.

## API Design

### JavaScript Interface

```javascript
import sftp from "k6/x/sftp";

export default function () {
  // connect() returns a Connection object
  const conn = sftp.connect(host, user, pass, port);

  // Methods are called on the connection
  conn.upload(data, "/remote/path");
  const files = conn.ls("/remote/dir");
  conn.download("/remote/file", "./local/file");

  // Always close when done
  conn.close();
}
```

### Method Signatures

| Method            | Parameters               | Returns           | Description                     |
| ----------------- | ------------------------ | ----------------- | ------------------------------- |
| `sftp.connect()`  | host, user, pass, port   | Connection        | Establishes SSH+SFTP connection |
| `conn.upload()`   | data (bytes), remotePath | error             | Writes data to remote file      |
| `conn.download()` | remotePath, localPath    | error             | Copies remote file to local     |
| `conn.ls()`       | path                     | []FileInfo, error | Lists directory contents        |
| `conn.close()`    | —                        | error             | Closes both SFTP and SSH        |

### FileInfo Object

The `ls()` method returns an array of objects:

```javascript
{
    name: "file.txt",    // File or directory name
    size: 1024,          // Size in bytes
    isDir: false,        // true if directory
    modTime: 1706000000  // Unix timestamp
}
```

## Error Handling

### Go Side

All methods return errors rather than panicking:

```go
func (c *Connection) Upload(data []byte, path string) error {
    if c.sftpClient == nil {
        return errors.New("not connected")
    }
    // ... operation ...
    if err != nil {
        return fmt.Errorf("open remote file: %w", err)
    }
    return nil
}
```

Errors are wrapped with context using `fmt.Errorf("context: %w", err)` to preserve the error chain.

### JavaScript Side

k6 scripts can handle errors with try/catch:

```javascript
export default function () {
  try {
    const conn = sftp.connect(host, user, pass, port);
    conn.upload(data, "/path");
    conn.close();
  } catch (e) {
    console.error(`SFTP error: ${e.message}`);
  }
}
```

Or use k6's `check()` for assertions in load tests.

## Resource Management

### Connection Lifecycle

1. **Creation:** `Connect()` creates both SSH and SFTP clients
2. **Usage:** Methods use the stored `sftpClient`
3. **Cleanup:** `Close()` closes SFTP first, then SSH

```go
func (c *Connection) Close() error {
    var errs []error

    if c.sftpClient != nil {
        if err := c.sftpClient.Close(); err != nil {
            errs = append(errs, err)
        }
        c.sftpClient = nil
    }

    if c.sshClient != nil {
        if err := c.sshClient.Close(); err != nil {
            errs = append(errs, err)
        }
        c.sshClient = nil
    }

    return errors.Join(errs...)
}
```

### File Handles

All file operations use `defer` for cleanup:

```go
func (c *Connection) Download(remotePath, localPath string) error {
    srcFile, err := c.sftpClient.Open(remotePath)
    if err != nil {
        return err
    }
    defer srcFile.Close()  // Always closes

    dstFile, err := os.Create(localPath)
    if err != nil {
        return err
    }
    defer dstFile.Close()  // Always closes

    _, err = io.Copy(dstFile, srcFile)
    return err
}
```

### Connection Timeouts

TCP and SSH connections have timeouts to prevent hanging:

```go
// TCP connection: 10 second timeout
netConn, err := net.DialTimeout("tcp", addr, 10*time.Second)

// SSH handshake: 30 second timeout
config := &ssh.ClientConfig{
    Timeout: 30 * time.Second,
    // ...
}
```

## Testing

### Unit Tests

Run with:

```bash
go test -v -short ./...
```

| Test                                     | Purpose                                           |
| ---------------------------------------- | ------------------------------------------------- |
| `TestConnection_NotConnected`            | Verifies methods return errors when not connected |
| `TestConnection_Close`                   | Verifies Close handles nil clients gracefully     |
| `TestClient_Connect_InvalidHost`         | Verifies connection errors are returned           |
| `TestModule_NewModuleInstance`           | Verifies module instantiation                     |
| `TestClient_Exports`                     | Verifies JavaScript exports                       |

### Concurrency Tests

These tests verify thread safety:

```bash
go test -race -v ./...
```

| Test                                | What It Does                                    |
| ----------------------------------- | ----------------------------------------------- |
| `TestConcurrency_MultipleClients`   | 10 goroutines × 100 iterations creating Clients |
| `TestConcurrency_ConnectionMethods` | 10 goroutines × 100 iterations calling methods  |

### Integration Tests

Require a real SFTP server. Set environment variables:

```bash
export SFTP_TEST_HOST=localhost
export SFTP_TEST_USER=testuser
export SFTP_TEST_PASS=testpass
go test -v ./...
```

Or use Docker:

```bash
docker run -p 2222:22 -d atmoz/sftp testuser:testpass:::upload
SFTP_TEST_HOST=localhost SFTP_TEST_PORT=2222 \
SFTP_TEST_USER=testuser SFTP_TEST_PASS=testpass \
go test -v ./...
```

### Race Detection

Build k6 with race detection:

```bash
xk6 build --race-detector --with xk6-sftp=.
./k6 run --vus 10 examples/xk6-sftp-01-file-upload.js
```

## Building

### With xk6 (Native)

```bash
# Install xk6
go install go.k6.io/xk6/cmd/xk6@latest

# Build k6 with extension
xk6 build --with xk6-sftp=.

# Build with race detector (for testing)
xk6 build --race-detector --with xk6-sftp=.
```

### With Docker (Cross-Compilation)

When building on Linux/Docker for macOS, the binary requires code signing before it will run.

```bash
# Build for macOS (Intel)
docker run --rm -u "$(id -u):$(id -g)" \
  -v "${PWD}:/xk6" -w /xk6 \
  -e GOOS=darwin -e GOARCH=amd64 \
  grafana/xk6 build --with xk6-sftp=.

# Build for macOS (Apple Silicon)
docker run --rm -u "$(id -u):$(id -g)" \
  -v "${PWD}:/xk6" -w /xk6 \
  -e GOOS=darwin -e GOARCH=arm64 \
  grafana/xk6 build --with xk6-sftp=.

# Build for Linux
docker run --rm -u "$(id -u):$(id -g)" \
  -v "${PWD}:/xk6" -w /xk6 \
  -e GOOS=linux -e GOARCH=amd64 \
  grafana/xk6 build --with xk6-sftp=.
```

### macOS Code Signing

Cross-compiled macOS binaries fail Gatekeeper's strict validation. After building with Docker, sign the binary:

```bash
# Ad-hoc sign (no Apple Developer ID required)
codesign -f -s - ./k6

# Verify signing
codesign -v ./k6

# Run
./k6 version
```

Without signing, macOS kills the process immediately:
```
[1]    12345 killed     ./k6 version
```

### Local Development

```bash
# Verify compilation
go build .

# Run tests
go test -v -short ./...

# Run tests with race detector
go test -race -v ./...
```

## Security Considerations

### Host Key Verification

Currently disabled for testing convenience:

```go
HostKeyCallback: ssh.InsecureIgnoreHostKey()
```

**For production use**, implement proper host key verification:

```go
HostKeyCallback: ssh.FixedHostKey(expectedKey)
// or
HostKeyCallback: knownhosts.New("~/.ssh/known_hosts")
```

### Authentication

Currently supports password authentication only. SSH key authentication is a planned enhancement.

### Credentials in Scripts

Avoid hardcoding credentials. Use environment variables:

```javascript
const host = __ENV.SFTP_HOST;
const user = __ENV.SFTP_USER;
const pass = __ENV.SFTP_PASS;
```

Run with:

```bash
./k6 run script.js -e SFTP_HOST=server -e SFTP_USER=user -e SFTP_PASS=pass
```
