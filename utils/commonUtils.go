package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SendOutput writes to the output channel without blocking.
func SendOutput(output chan<- string, msg string) {
	if output == nil {
		return
	}
	select {
	case output <- msg:
	default:
	}
}

// LocalZip 本地压缩文件夹为ZIP格式
func LocalZip(dirPath, outFullName string) error {
	zipFile, err := os.Create(outFullName)
	if err != nil {
		return fmt.Errorf("创建ZIP文件失败: %v", err)
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	base, err := filepath.Abs(filepath.Clean(dirPath))
	if err != nil {
		return fmt.Errorf("解析目录路径失败: %w", err)
	}

	err = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 跳过目录（只处理文件）
		if !info.Mode().IsRegular() {
			return nil
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("解析文件路径失败: %w", err)
		}
		rel, err := filepath.Rel(base, absPath)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %w", err)
		}
		if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
			return fmt.Errorf("路径越界: %s", path)
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("创建ZIP头信息失败: %v", err)
		}
		// 使用正斜杠，避免 Windows 下 zip 内带盘符路径，Linux unzip 错建成 D:\... 目录
		header.Name = filepath.ToSlash(rel)
		header.Method = zip.Deflate
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("创建ZIP条目失败: %v", err)
		}
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("打开文件失败: %v", err)
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		if err != nil {
			return fmt.Errorf("写入ZIP文件失败: %v", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("遍历目录并压缩文件时出错: %v", err)
	}
	return nil
}
