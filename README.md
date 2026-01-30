# xk6-sftp - SFTP operations for k6

A [k6](https://github.com/grafana/k6) extension for interacting with SFTP servers.

## Build

### Docker

The below steps are sourced from the official [k6 documentation](https://grafana.com/docs/k6/latest/extensions/build-k6-binary-using-docker/)
on this topic. If the following steps contain errors, please reference the official documentation.

```bash
docker run --rm -u "$(id -u):$(id -g)" -v "${PWD}:/xk6" grafana/xk6 build --with github.com/grafana/xk6-sftp@latest
```

### Local (via `go`)

#### Prerequisites

To build a `k6` binary with this extension, first ensure you have the following:

- [Go Toolchain](https://go.dev/doc/toolchain)
- Git

#### Steps

1. Download [xk6](https://github.com/grafana/xk6/):

```bash
go install github.com/grafana/xk6/cmd/xk6@latest
```

2. Build the binary:

```bash
xk6 build --with github.com/grafana/xk6-sftp@latest
```

This will result in a `k6` binary in the current directory.

## Usage

```javascript
import sftp from "k6/x/sftp";

export default function () {
  // connect() returns a Connection object that is unique to the VU
  const conn = sftp.connect(
    __ENV.SFTP_HOST,
    __ENV.SFTP_USER,
    __ENV.SFTP_PASS,
    parseInt(__ENV.SFTP_PORT) || 22,
  );

  // Upload a file
  const data = open("myfile.txt");
  conn.upload(data, "/remote/path/myfile.txt");

  // List directory contents
  const files = conn.ls("/remote/path");
  files.forEach((f) => console.log(`${f.name} - ${f.size} bytes`));

  // Download a file
  conn.download("/remote/path/file.txt", "./local-file.txt");

  // Always close the connection
  conn.close();
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
