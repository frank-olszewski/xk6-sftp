package sftp

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/pkg/sftp"
	"go.k6.io/k6/js/modules"
	"golang.org/x/crypto/ssh"
)

func init() {
	modules.Register("k6/x/sftp", new(Sftp))
}

type Sftp struct {
	FileSubmitted bool
	Status        string
	FilePresence  bool
	Client        *ssh.Client
	error         error
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

// Connect to the remote SFTP server
func (s *Sftp) Connect(host, username, password string, port int) error {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For testing purposes only, not for production
	}
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	checkError(err)

	s.Client = conn
	return nil
}

// Disconnect from the active SFTP instance, if one exists
func (s *Sftp) Disconnect() error {
	if s.Client == nil {
		return nil
	}

	err := s.Client.Close()
	checkError(err)

	return err
}

// Upload file to the active SFTP instance
func (s *Sftp) Upload(srcbytes []byte, dst string) error {
	if s.Client == nil {
		return nil
	}
	client, err := sftp.NewClient(s.Client)
	checkError(err)

	defer client.Close()

	file, err := client.OpenFile(dst, os.O_CREATE)
	checkError(err)
	defer file.Close()

	if _, err := file.Write(srcbytes); err != nil {
		return err
	}

	return nil
}

// Ls is used to list all files and directories in the defined path
// Returns an array of https://pkg.go.dev/io/fs#FileInfo objects
func (s *Sftp) Ls(path string) ([]os.FileInfo, error) {
	if s.Client == nil {
		return nil, nil
	}
	client, err := sftp.NewClient(s.Client)
	checkError(err)
	defer client.Close()

	fileInfoResults, err := client.ReadDir(path)
	checkError(err)

	for _, fileInfoResult := range fileInfoResults {
		fileInfoResult.Name()
	}
	return fileInfoResults, nil
}

// Download the remote file to a local file
func (s *Sftp) Download(path, dst string) error {
	if s.Client == nil {
		return nil
	}
	client, err := sftp.NewClient(s.Client)
	checkError(err)

	srcfile, err := client.OpenFile(path, os.O_RDONLY)
	checkError(err)
	dstfile, err := os.Create(dst)
	checkError(err)

	reader := bufio.NewReader(srcfile)
	checkError(err)
	writer := bufio.NewWriter(dstfile)
	checkError(err)
	defer writer.Flush()

	buffer := make([]byte, 1024) // 1024 byte-sized buffer
	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break // EOF
		}
		if _, err := writer.Write(buffer[:n]); err != nil {
			return err
		}
	}

	return nil
}
