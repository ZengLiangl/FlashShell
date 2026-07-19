package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const externalEditEvent = "shell:external-edit"

type externalEditWatch struct {
	machineName  string
	remotePath   string
	localPath    string
	baselineMod  time.Time
	baselineSize int64
	lastMod      time.Time
	lastSize     int64
	uploading    bool
	cancel       context.CancelFunc
}

type externalEditStore struct {
	mu    sync.Mutex
	items map[string]*externalEditWatch
}

func newExternalEditStore() *externalEditStore {
	return &externalEditStore{items: make(map[string]*externalEditWatch)}
}

func externalEditKey(machineName, remotePath string) string {
	return machineName + "\x00" + remotePath
}

func (s *externalEditStore) stop(key string) {
	s.mu.Lock()
	w := s.items[key]
	delete(s.items, key)
	s.mu.Unlock()
	if w != nil && w.cancel != nil {
		w.cancel()
	}
}

func (s *externalEditStore) stopForMachine(machineName string) {
	s.mu.Lock()
	var keys []string
	for k, w := range s.items {
		if w.machineName == machineName {
			keys = append(keys, k)
		}
	}
	s.mu.Unlock()
	for _, k := range keys {
		s.stop(k)
	}
}

func (a *App) emitExternalEdit(machineName, remotePath, status, message string) {
	if a.ctx == nil {
		return
	}
	wailsRuntime.EventsEmit(a.ctx, externalEditEvent, map[string]string{
		"machineName": machineName,
		"remotePath":  remotePath,
		"status":      status,
		"message":     message,
	})
}

func (a *App) startExternalEditWatch(machineName, remotePath, localPath string) error {
	info, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	if a.externalEdits == nil {
		a.externalEdits = newExternalEditStore()
	}
	key := externalEditKey(machineName, remotePath)
	a.externalEdits.stop(key)

	ctx, cancel := context.WithCancel(context.Background())
	w := &externalEditWatch{
		machineName:  machineName,
		remotePath:   remotePath,
		localPath:    localPath,
		baselineMod:  info.ModTime(),
		baselineSize: info.Size(),
		lastMod:      info.ModTime(),
		lastSize:     info.Size(),
		cancel:       cancel,
	}
	a.externalEdits.mu.Lock()
	a.externalEdits.items[key] = w
	a.externalEdits.mu.Unlock()

	go a.runExternalEditWatch(ctx, w)
	return nil
}

func (a *App) runExternalEditWatch(ctx context.Context, w *externalEditWatch) {
	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()
	var debounce *time.Timer

	upload := func() {
		if w.uploading {
			return
		}
		info, err := os.Stat(w.localPath)
		if err != nil {
			return
		}
		if info.ModTime().Equal(w.baselineMod) && info.Size() == w.baselineSize {
			return
		}
		if info.ModTime().Equal(w.lastMod) && info.Size() == w.lastSize {
			return
		}
		w.uploading = true
		go func() {
			defer func() { w.uploading = false }()
			aux, err := a.getShellAux(w.machineName)
			if err != nil {
				a.emitExternalEdit(w.machineName, w.remotePath, "error", err.Error())
				return
			}
			remotePath := w.remotePath
			if err := aux.UploadFile(context.Background(), w.localPath, remotePath, nil); err != nil {
				a.emitExternalEdit(w.machineName, w.remotePath, "error", err.Error())
				return
			}
			if st, err := os.Stat(w.localPath); err == nil {
				w.lastMod = st.ModTime()
				w.lastSize = st.Size()
			}
			name := filepath.Base(remotePath)
			a.emitExternalEdit(w.machineName, w.remotePath, "uploaded", fmt.Sprintf("已上传 %s", name))
		}()
	}

	for {
		select {
		case <-ctx.Done():
			if debounce != nil {
				debounce.Stop()
			}
			return
		case <-ticker.C:
			info, err := os.Stat(w.localPath)
			if err != nil {
				continue
			}
			if info.ModTime().Equal(w.baselineMod) && info.Size() == w.baselineSize {
				continue
			}
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(900*time.Millisecond, upload)
		}
	}
}
