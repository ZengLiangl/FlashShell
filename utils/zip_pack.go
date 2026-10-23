package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalZipSources 将多个本地文件或目录打包为一个 ZIP（目录规则与 LocalZip 一致：目录内文件以相对目录根的路径写入）。
func LocalZipSources(outFullName string, sources []string) error {
	if len(sources) == 0 {
		return fmt.Errorf("至少需要一个源路径")
	}
	outFullName = filepath.Clean(outFullName)
	if err := os.MkdirAll(filepath.Dir(outFullName), 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	zipFile, err := os.Create(outFullName)
	if err != nil {
		return fmt.Errorf("创建ZIP文件失败: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, src := range sources {
		src = strings.TrimSpace(src)
		if src == "" {
			continue
		}
		info, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("源路径不可用 %s: %w", src, err)
		}
		if info.Mode().IsRegular() {
			if err := addZipFile(zipWriter, src, filepath.Base(src)); err != nil {
				return err
			}
			continue
		}
		if info.IsDir() {
			base, err := filepath.Abs(src)
			if err != nil {
				return fmt.Errorf("解析目录路径失败: %w", err)
			}
			err = filepath.Walk(base, func(path string, fi os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if !fi.Mode().IsRegular() {
					return nil
				}
				absPath, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(base, absPath)
				if err != nil {
					return err
				}
				if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
					return fmt.Errorf("路径越界: %s", path)
				}
				return addZipFile(zipWriter, path, filepath.ToSlash(rel))
			})
			if err != nil {
				return fmt.Errorf("压缩目录 %s 失败: %w", src, err)
			}
			continue
		}
		return fmt.Errorf("不支持的源类型: %s", src)
	}
	return nil
}

func addZipFile(zipWriter *zip.Writer, filePath, entryName string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("创建ZIP头信息失败: %w", err)
	}
	header.Name = filepath.ToSlash(entryName)
	header.Method = zip.Store
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("创建ZIP条目失败: %w", err)
	}
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(writer, file); err != nil {
		return fmt.Errorf("写入ZIP失败: %w", err)
	}
	return nil
}
