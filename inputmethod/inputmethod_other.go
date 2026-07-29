//go:build !darwin && !windows

package inputmethod

import "fmt"

func platformEnter() error {
	return fmt.Errorf("当前平台不支持临时关闭中文组词")
}

func platformLeave() error {
	return nil
}
