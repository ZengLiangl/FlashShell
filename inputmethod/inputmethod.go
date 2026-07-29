// Package inputmethod 在 Shell 终端焦点期间临时关闭中文组词，失焦后恢复。
package inputmethod

import "sync"

var (
	mu      sync.Mutex
	engaged bool
	busy    bool
)

// Enter 进入英文输入态（关闭中文组词）。可重复调用，已生效时为 no-op。
// 注意：平台 API（尤其是 macOS TIS）可能同步切到主线程，故不在持锁期间调用。
func Enter() {
	mu.Lock()
	if engaged || busy {
		mu.Unlock()
		return
	}
	busy = true
	mu.Unlock()

	err := platformEnter()

	mu.Lock()
	busy = false
	if err == nil {
		engaged = true
	}
	mu.Unlock()
}

// Leave 恢复进入前的输入法状态。可重复调用。
func Leave() {
	mu.Lock()
	if !engaged || busy {
		mu.Unlock()
		return
	}
	busy = true
	engaged = false
	mu.Unlock()

	_ = platformLeave()

	mu.Lock()
	busy = false
	mu.Unlock()
}

// Engaged 当前是否处于临时英文输入态。
func Engaged() bool {
	mu.Lock()
	defer mu.Unlock()
	return engaged
}
