package cmds

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"FlashDock/define"
	"FlashDock/utils"
	"strings"
)

func RegTarCmd() {
	CmdManager.RegSpecialCmd("targz", doTar)
}
func doTar(rm *define.RemoteMachine, c []string, outputChan chan<- string) error {
	if len(c) != 3 {
		return errors.New("参数错误" + strings.Join(c, ","))
	}
	src := c[1]
	dest := c[2]
	err := CreateTarGz(src, dest)
	if err != nil {
		utils.SendOutput(outputChan, fmt.Sprintf("压缩失败: %s", err.Error()))
	} else {
		
	}
	return err
}

// CreateTarGz 将指定目录下的所有文件打包成 .tar.gz 文件
func CreateTarGz(src string, dst string) error {
	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tarWriter := tar.NewWriter(gzw)
	defer tarWriter.Close()

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// 将相对路径转换为绝对路径，以便在解压时正确还原
		header.Name = filepath.ToSlash(filepath.Clean(strings.TrimPrefix(path, src)))

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})
}
