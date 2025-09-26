package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalZip 本地压缩文件夹为ZIP格式
func LocalZip(dirPath, outFullName string) error {
	zipFile, err := os.Create(outFullName)
	if err != nil {
		return fmt.Errorf("创建ZIP文件失败: %v", err)
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 跳过目录（只处理文件）
		if !info.Mode().IsRegular() {
			return nil
		}
		relativePath := strings.TrimPrefix(path, dirPath+string(os.PathSeparator))
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("创建ZIP头信息失败: %v", err)
		}
		header.Name = relativePath // 设置相对路径
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
