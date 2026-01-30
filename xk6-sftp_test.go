package sftp

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestConnection_NotConnected verifies that all Connection methods
// return appropriate errors when the connection is not established
func TestConnection_NotConnected(t *testing.T) {
	conn := &Connection{
		sshClient:  nil,
		sftpClient: nil,
	}

	t.Run("Upload returns error when not connected", func(t *testing.T) {
		err := conn.Upload([]byte("test data"), "/remote/path")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if err.Error() != "not connected" {
			t.Errorf("expected 'not connected' error, got: %v", err)
		}
	})

	t.Run("Download returns error when not connected", func(t *testing.T) {
		err := conn.Download("/remote/path", "/local/path")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if err.Error() != "not connected" {
			t.Errorf("expected 'not connected' error, got: %v", err)
		}
	})

	t.Run("Ls returns error when not connected", func(t *testing.T) {
		files, err := conn.Ls("/remote/path")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if err.Error() != "not connected" {
			t.Errorf("expected 'not connected' error, got: %v", err)
		}
		if files != nil {
			t.Error("expected nil files, got non-nil")
		}
	})
}

// TestConnection_Close verifies Close behavior
func TestConnection_Close(t *testing.T) {
	t.Run("Close on nil connection succeeds", func(t *testing.T) {
		conn := &Connection{
			sshClient:  nil,
			sftpClient: nil,
		}
		err := conn.Close()
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("Close sets clients to nil", func(t *testing.T) {
		conn := &Connection{
			sshClient:  nil,
			sftpClient: nil,
		}
		_ = conn.Close()
		if conn.sshClient != nil {
			t.Error("expected sshClient to be nil after Close")
		}
		if conn.sftpClient != nil {
			t.Error("expected sftpClient to be nil after Close")
		}
	})
}

// TestClient_Connect_InvalidHost verifies connection error handling
func TestClient_Connect_InvalidHost(t *testing.T) {
	c := &Client{}

	t.Run("Connect to localhost invalid port returns error quickly", func(t *testing.T) {
		// localhost with a port that shouldn't have SSH - fails fast with connection refused
		conn, err := c.Connect("127.0.0.1", "user", "pass", 65534)
		if err == nil {
			if conn != nil {
				conn.Close()
			}
			t.Error("expected error for invalid port, got nil")
		}
		if conn != nil {
			t.Error("expected nil connection on error")
		}
	})

	t.Run("Connect to unreachable host returns error", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping slow network test in short mode")
		}
		// Use a non-routable IP - this will timeout but now has a 10s limit
		conn, err := c.Connect("192.0.2.1", "user", "pass", 22)
		if err == nil {
			if conn != nil {
				conn.Close()
			}
			t.Error("expected error for unreachable host, got nil")
		}
		if conn != nil {
			t.Error("expected nil connection on error")
		}
	})
}

// TestModule_NewModuleInstance verifies module instantiation
func TestModule_NewModuleInstance(t *testing.T) {
	m := &Module{}

	t.Run("NewModuleInstance returns non-nil instance", func(t *testing.T) {
		// Pass nil for VU since we're just testing the instance creation
		instance := m.NewModuleInstance(nil)
		if instance == nil {
			t.Error("expected non-nil instance")
		}

		c, ok := instance.(*Client)
		if !ok {
			t.Error("expected instance to be *Client")
		}
		if c == nil {
			t.Error("expected non-nil Client")
		}
	})
}

// TestClient_Exports verifies the exported functions
func TestClient_Exports(t *testing.T) {
	c := &Client{}
	exports := c.Exports()

	t.Run("Exports contains connect function", func(t *testing.T) {
		if exports.Named == nil {
			t.Fatal("expected Named exports, got nil")
		}

		connectFn, exists := exports.Named["connect"]
		if !exists {
			t.Error("expected 'connect' in Named exports")
		}
		if connectFn == nil {
			t.Error("expected non-nil connect function")
		}
	})

	t.Run("Default export is nil", func(t *testing.T) {
		if exports.Default != nil {
			t.Error("expected nil Default export")
		}
	})
}

// TestConcurrency_MultipleClients verifies that multiple goroutines
// can safely create and use their own Clients and connections
// This simulates k6's VU behavior where each VU has its own Client
func TestConcurrency_MultipleClients(t *testing.T) {
	m := &Module{}
	numGoroutines := 10
	iterations := 100

	// Use a WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Channel to collect any errors
	errCh := make(chan error, numGoroutines*iterations)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Each goroutine gets its own Client (like a k6 VU)
			c := m.NewModuleInstance(nil).(*Client)

			for j := 0; j < iterations; j++ {
				// Verify exports are accessible
				exports := c.Exports()
				if exports.Named["connect"] == nil {
					errCh <- fmt.Errorf("goroutine %d: connect is nil", id)
					return
				}

				// Create a connection (will fail but tests the code path)
				conn, err := c.Connect("127.0.0.1", "user", "pass", 65534)
				if err == nil {
					// If somehow it connected, close it
					conn.Close()
				}
				// Error is expected - we just want to verify no race
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	// Check for any errors
	for err := range errCh {
		t.Error(err)
	}
}

// TestConcurrency_ConnectionMethods verifies that Connection methods
// can be called safely (they should return errors since not connected)
func TestConcurrency_ConnectionMethods(t *testing.T) {
	numGoroutines := 10
	iterations := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			// Each goroutine creates its own connection struct
			conn := &Connection{
				sshClient:  nil,
				sftpClient: nil,
			}

			for j := 0; j < iterations; j++ {
				// Call all methods - they should return "not connected" errors
				_ = conn.Upload([]byte("data"), "/path")
				_ = conn.Download("/remote", "/local")
				_, _ = conn.Ls("/path")
				_ = conn.Close()
			}
		}()
	}

	wg.Wait()
}

// TestConnection_UploadDownload_Integration is a placeholder for integration tests
// These require a real SFTP server (see xk6-sftp-12)
func TestConnection_Integration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SFTP_TEST_HOST") == "" {
		t.Skip("Skipping integration tests: SFTP_TEST_HOST not set")
	}

	host := os.Getenv("SFTP_TEST_HOST")
	user := os.Getenv("SFTP_TEST_USER")
	pass := os.Getenv("SFTP_TEST_PASS")
	port := 22

	c := &Client{}
	conn, err := c.Connect(host, user, pass, port)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	t.Run("Upload and verify", func(t *testing.T) {
		testData := []byte("test content from unit test")
		remotePath := "/upload/test-unit.txt"

		err := conn.Upload(testData, remotePath)
		if err != nil {
			t.Errorf("Upload failed: %v", err)
		}
	})

	t.Run("Ls directory", func(t *testing.T) {
		files, err := conn.Ls("/upload")
		if err != nil {
			t.Errorf("Ls failed: %v", err)
		}
		if files == nil {
			t.Error("expected files, got nil")
		}

		// Verify file info structure
		for _, f := range files {
			if _, ok := f["name"]; !ok {
				t.Error("file info missing 'name' field")
			}
			if _, ok := f["size"]; !ok {
				t.Error("file info missing 'size' field")
			}
			if _, ok := f["isDir"]; !ok {
				t.Error("file info missing 'isDir' field")
			}
			if _, ok := f["modTime"]; !ok {
				t.Error("file info missing 'modTime' field")
			}
		}
	})

	t.Run("Download file", func(t *testing.T) {
		remotePath := "/upload/test-unit.txt"
		localPath := filepath.Join(t.TempDir(), "downloaded.txt")

		err := conn.Download(remotePath, localPath)
		if err != nil {
			t.Errorf("Download failed: %v", err)
		}

		// Verify file exists and has content
		data, err := os.ReadFile(localPath)
		if err != nil {
			t.Errorf("Failed to read downloaded file: %v", err)
		}
		if string(data) != "test content from unit test" {
			t.Errorf("Downloaded content mismatch: got %q", string(data))
		}
	})
}
