package machine

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	transferPoolMaxNormal = 4
	transferPoolMaxSudo   = 2
	transferPoolIdleTTL   = 8 * time.Second
)

type transferPoolSlot struct {
	client    *sftp.Client
	idleAt    time.Time
	idleTimer *time.Timer
}

// sftpTransferPool 在同一 SSH 上租用额外 SFTP 通道，供并发传输与浏览通道分离。
type sftpTransferPool struct {
	mu       sync.Mutex
	ssh      *ssh.Client
	sudo     bool
	password string
	max      int
	busy     int
	idle     []*transferPoolSlot
	waiters  []chan struct{}
	closed   bool
}

func newSFTPTransferPool(sshClient *ssh.Client, sudo bool, password string) *sftpTransferPool {
	max := transferPoolMaxNormal
	if sudo {
		max = transferPoolMaxSudo
	}
	return &sftpTransferPool{
		ssh:      sshClient,
		sudo:     sudo,
		password: password,
		max:      max,
	}
}

func (p *sftpTransferPool) Acquire() (*sftp.Client, error) {
	for {
		p.mu.Lock()
		if p.closed || p.ssh == nil {
			p.mu.Unlock()
			return nil, fmt.Errorf("传输通道池已关闭")
		}
		if len(p.idle) > 0 {
			slot := p.idle[len(p.idle)-1]
			p.idle = p.idle[:len(p.idle)-1]
			if slot.idleTimer != nil {
				slot.idleTimer.Stop()
				slot.idleTimer = nil
			}
			p.busy++
			c := slot.client
			p.mu.Unlock()
			return c, nil
		}
		if p.busy < p.max {
			p.busy++
			sshClient := p.ssh
			sudo := p.sudo
			password := p.password
			p.mu.Unlock()
			c, err := OpenSFTPClient(sshClient, sudo, password)
			if err != nil {
				p.mu.Lock()
				p.busy--
				p.signalWaiterLocked()
				p.mu.Unlock()
				return nil, err
			}
			return c, nil
		}
		ch := make(chan struct{}, 1)
		p.waiters = append(p.waiters, ch)
		p.mu.Unlock()
		<-ch
	}
}

func (p *sftpTransferPool) Release(c *sftp.Client) {
	if c == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		_ = c.Close()
		return
	}
	if p.busy > 0 {
		p.busy--
	}
	slot := &transferPoolSlot{client: c, idleAt: time.Now()}
	slot.idleTimer = time.AfterFunc(transferPoolIdleTTL, func() {
		p.evictIdle(c)
	})
	p.idle = append(p.idle, slot)
	p.signalWaiterLocked()
}

func (p *sftpTransferPool) Discard(c *sftp.Client) {
	if c == nil {
		return
	}
	_ = c.Close()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.busy > 0 {
		p.busy--
	}
	p.signalWaiterLocked()
}

func (p *sftpTransferPool) evictIdle(c *sftp.Client) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, slot := range p.idle {
		if slot.client == c {
			p.idle = append(p.idle[:i], p.idle[i+1:]...)
			if slot.idleTimer != nil {
				slot.idleTimer.Stop()
			}
			_ = c.Close()
			return
		}
	}
}

func (p *sftpTransferPool) signalWaiterLocked() {
	if len(p.waiters) == 0 {
		return
	}
	ch := p.waiters[0]
	p.waiters = p.waiters[1:]
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (p *sftpTransferPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	for _, slot := range p.idle {
		if slot.idleTimer != nil {
			slot.idleTimer.Stop()
		}
		_ = slot.client.Close()
	}
	p.idle = nil
	for _, ch := range p.waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	p.waiters = nil
}
