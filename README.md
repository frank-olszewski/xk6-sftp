# xk6-sftp - SFTP operations for k6 
A [k6](https://github.com/grafana/k6) extension for interacting with SFTP servers.

## Build
### Docker
The below steps are sourced from the official [k6 documentation](https://grafana.com/docs/k6/latest/extensions/build-k6-binary-using-docker/)
on this topic.  If the following steps contain errors, please reference the official documentation.
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
import ssh from 'k6/x/ssh';

export default function () {
  ssh.connect({
    username: `${__ENV.K6_USERNAME}`,
    password: `${__ENV.K6_PASSWORD}`,
    host: [HOSTNAME],
	port: 22
  })
  console.log(ssh.run('pwd'))
}
```

For more specific examples, please check the `examples/` subdirectory.

## Testing locally
To be further expanded upon at a later date.

`$ ./k6 run examples/xk6-sftp-03-download-remote-file.js -e SFTP_HOST=${HOST} -e SFTP_PORT=${PORT} -e SFTP_USER=${USER} -e SFTP_PASS=${PASS}`

## Frequently asked questions
