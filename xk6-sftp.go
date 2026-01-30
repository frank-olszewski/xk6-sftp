package sftp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/pkg/sftp"
	"go.k6.io/k6/js/modules"
	"golang.org/x/crypto/ssh"
)

func init() {
	modules.Register("k6/x/sftp", new(Module))
}

// Module is the root-level module registered with k6
type Module struct{}

// NewModuleInstance creates a Client for each VU
func (*Module) NewModuleInstance(vu modules.VU) modules.Instance {
	return &Client{}
}

// Client represents the SFTP client for a single VU
type Client struct{}

// Exports returns the exports of the module for JavaScript
func (c *Client) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"connect": c.Connect,
		},
	}
}

// Connection represents a single SFTP connection
// Each VU gets its own Connection instance, avoiding shared state
type Connection struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

// Connect establishes an SSH connection and creates an SFTP client
// Returns a Connection that the caller owns and must close
func (c *Client) Connect(host, username, password string, port int) (*Connection, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For testing purposes only
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	// Use a dialer with timeout for the TCP connection
	netConn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("tcp dial failed: %w", err)
	}

	// Establish SSH connection over the TCP connection
	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, config)
	if err != nil {
		netConn.Close()
		return nil, fmt.Errorf("ssh handshake failed: %w", err)
	}

	sshClient := ssh.NewClient(sshConn, chans, reqs)

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close() // Clean up SSH if SFTP fails
		return nil, fmt.Errorf("sftp client creation failed: %w", err)
	}

	return &Connection{
		sshClient:  sshClient,
		sftpClient: sftpClient,
	}, nil
}

// Close closes both the SFTP and SSH connections
func (c *Connection) Close() error {
	var errs []error

	if c.sftpClient != nil {
		if err := c.sftpClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("sftp close: %w", err))
		}
		c.sftpClient = nil
	}

	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("ssh close: %w", err))
		}
		c.sshClient = nil
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Upload writes data to a remote file
func (c *Connection) Upload(data []byte, remotePath string) error {
	if c.sftpClient == nil {
		return errors.New("not connected")
	}

	file, err := c.sftpClient.OpenFile(remotePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("open remote file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write to remote file: %w", err)
	}

	return nil
}

// Download copies a remote file to a local path
func (c *Connection) Download(remotePath, localPath string) error {
	if c.sftpClient == nil {
		return errors.New("not connected")
	}

	srcFile, err := c.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open remote file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	return nil
}

// Ls lists files and directories at the given remote path
// Returns an array of objects with name, size, isDir, and modTime properties
func (c *Connection) Ls(path string) ([]map[string]interface{}, error) {
	if c.sftpClient == nil {
		return nil, errors.New("not connected")
	}

	entries, err := c.sftpClient.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	results := make([]map[string]interface{}, len(entries))
	for i, entry := range entries {
		results[i] = map[string]interface{}{
			"name":    entry.Name(),
			"size":    entry.Size(),
			"isDir":   entry.IsDir(),
			"modTime": entry.ModTime().Unix(),
		}
	}

	return results, nil
}
