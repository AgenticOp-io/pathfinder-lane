// Package sftpclient is a thin SFTP helper over an existing ssh.Client.
//
// The Fyne file-transfer dialog calls this; nothing here draws UI. Prefer SFTP
// over classic SCP — same job, supported by OpenSSH and network gear alike.
package sftpclient

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Entry is one remote directory row.
type Entry struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// Client wraps pkg/sftp.
type Client struct {
	c *sftp.Client
}

func Open(sshClient *ssh.Client) (*Client, error) {
	c, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, err
	}
	return &Client{c: c}, nil
}

func (c *Client) Close() error {
	if c == nil || c.c == nil {
		return nil
	}
	return c.c.Close()
}

func (c *Client) List(dir string) ([]Entry, error) {
	if dir == "" {
		dir = "."
	}
	fis, err := c.c.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(fis))
	for _, fi := range fis {
		out = append(out, Entry{
			Name:    fi.Name(),
			Path:    path.Join(dir, fi.Name()),
			IsDir:   fi.IsDir(),
			Size:    fi.Size(),
			ModTime: fi.ModTime(),
		})
	}
	return out, nil
}

func (c *Client) Download(remote, local string) error {
	rf, err := c.c.Open(remote)
	if err != nil {
		return err
	}
	defer rf.Close()
	lf, err := os.Create(local)
	if err != nil {
		return err
	}
	defer lf.Close()
	_, err = io.Copy(lf, rf)
	return err
}

func (c *Client) Upload(local, remote string) error {
	lf, err := os.Open(local)
	if err != nil {
		return err
	}
	defer lf.Close()
	rf, err := c.c.Create(remote)
	if err != nil {
		return err
	}
	defer rf.Close()
	_, err = io.Copy(rf, lf)
	return err
}

func (c *Client) Mkdir(remote string) error {
	return c.c.Mkdir(remote)
}

func (c *Client) Remove(remote string) error {
	return c.c.Remove(remote)
}

func (c *Client) Rename(oldpath, newpath string) error {
	return c.c.Rename(oldpath, newpath)
}

func Join(elem ...string) string {
	return path.Join(elem...)
}

func EnsureDir(c *Client, remote string) error {
	if remote == "" || remote == "." || remote == "/" {
		return nil
	}
	if _, err := c.c.Stat(remote); err == nil {
		return nil
	}
	if err := EnsureDir(c, path.Dir(remote)); err != nil {
		return err
	}
	if err := c.c.Mkdir(remote); err != nil {
		return fmt.Errorf("mkdir %s: %w", remote, err)
	}
	return nil
}
