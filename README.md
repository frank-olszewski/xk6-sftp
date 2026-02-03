# xk6-sftp - SFTP operations for k6

A [k6](https://github.com/grafana/k6) extension for interacting with SFTP servers.

## Build

### macOS (Recommended)

Native builds work best on macOS. Install the Go toolchain first, then:

```bash
# Install xk6
go install go.k6.io/xk6/cmd/xk6@latest

# Build k6 with this extension
xk6 build --with github.com/grafana/xk6-sftp@latest
```

### macOS via Docker

If you prefer Docker, cross-compiled binaries require code signing before macOS will run them:

```bash
# Build for Apple Silicon (M1/M2/M3)
docker run --rm -u "$(id -u):$(id -g)" \
  -v "${PWD}:/xk6" -w /xk6 \
  -e GOOS=darwin -e GOARCH=arm64 \
  grafana/xk6 build --with github.com/grafana/xk6-sftp@latest

# Build for Intel Mac
docker run --rm -u "$(id -u):$(id -g)" \
  -v "${PWD}:/xk6" -w /xk6 \
  -e GOOS=darwin -e GOARCH=amd64 \
  grafana/xk6 build --with github.com/grafana/xk6-sftp@latest

# When building a locally-developed xk6-sftp project
docker run --rm -u "$(id -u):$(id -g)" \
  -v "${PWD}:/xk6" -w /xk6 \
  -e GOOS=darwin -e GOARCH=arm64 \
  grafana/xk6 build --with xk6-sftp=.

# Sign the binary (required for cross-compiled macOS binaries)
codesign -f -s - ./k6
```

Without signing, macOS kills the binary immediately with no error message.

### Linux

```bash
docker run --rm -u "$(id -u):$(id -g)" \
  -v "${PWD}:/xk6" -w /xk6 \
  grafana/xk6 build --with github.com/grafana/xk6-sftp@latest
```

### From Source

For local development:

```bash
# Clone the repo
git clone https://github.com/grafana/xk6-sftp.git
cd xk6-sftp

# Build with local changes
xk6 build --with xk6-sftp=.
```

See the [k6 documentation](https://grafana.com/docs/k6/latest/extensions/build-k6-binary-using-docker/) for more build options.

## Usage

```javascript
import sftp from "k6/x/sftp";

// Use binary mode ('b') for file uploads
const fileData = open("myfile.txt", "b");

export default function () {
  let conn;
  try {
    conn = sftp.connect(
      __ENV.SFTP_HOST,
      __ENV.SFTP_USER,
      __ENV.SFTP_PASS,
      parseInt(__ENV.SFTP_PORT) || 22,
    );

    // Upload a file
    conn.upload(fileData, "/remote/path/myfile.txt");

    // List directory contents
    const files = conn.ls("/remote/path");
    files.forEach((f) => console.log(`${f.name} - ${f.size} bytes`));

    // Download a file
    conn.download("/remote/path/file.txt", "./local-file.txt");
  } catch (err) {
    console.error(`SFTP error: ${err}`);
  } finally {
    if (conn) {
      conn.close();
    }
  }
}
```

For more specific examples, please check the `examples/` subdirectory.

## API Reference

### `sftp.connect(host, username, password, port)`

Establishes an SFTP connection and returns a `Connection` object.

- `host` (string): SFTP server hostname
- `username` (string): SSH username
- `password` (string): SSH password
- `port` (number): SSH port (typically 22)
- Returns: `Connection` object

### `conn.upload(data, remotePath)`

Uploads data to a remote file.

- `data` (ArrayBuffer): File contents to upload
- `remotePath` (string): Destination path on the remote server

### `conn.download(remotePath, localPath)`

Downloads a remote file to the local filesystem.

- `remotePath` (string): Path to file on remote server
- `localPath` (string): Destination path on local filesystem

### `conn.ls(path)`

Lists files and directories at the given path.

- `path` (string): Remote directory path
- Returns: Array of file info objects with properties:
  - `name` (string): File/directory name
  - `size` (number): Size in bytes
  - `isDir` (boolean): True if directory
  - `modTime` (number): Modification time (Unix timestamp)

### `conn.close()`

Closes the SFTP and SSH connections. Always call this when done.

## Testing locally

```bash
./k6 run examples/xk6-sftp-01-file-upload.js \
  -e SFTP_HOST=${HOST} \
  -e SFTP_PORT=${PORT} \
  -e SFTP_USER=${USER} \
  -e SFTP_PASS=${PASS}
```
