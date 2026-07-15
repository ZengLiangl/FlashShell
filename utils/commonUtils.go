package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
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
		// Store：上传链路本身已是瓶颈，CPU 再 Deflate 会进一步拖慢整体吞吐
		header.Method = zip.Store
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("创建ZIP条目失败: %v", err)
		}
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("打开文件失败: %v", err)
		}
		defer file.Close()
		_, err = CopyBuffer(writer, file)
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

// LocalUnzip 将 zip 解压到目标目录（防路径穿越）
func LocalUnzip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开ZIP失败: %w", err)
	}
	defer r.Close()

	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destAbs, 0o755); err != nil {
		return err
	}

	for _, f := range r.File {
		name := filepath.Clean(filepath.FromSlash(f.Name))
		if name == "." || name == "" {
			continue
		}
		target := filepath.Join(destAbs, name)
		rel, err := filepath.Rel(destAbs, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("非法ZIP路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		closeErr := out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

// LocalExtractTarGz 用纯 Go 解压 tar.gz（避免 Windows 系统 tar 把 C: 当成远程主机）
func LocalExtractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("打开压缩包失败: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("打开 gzip 失败: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destAbs, 0o755); err != nil {
		return err
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取 tar 失败: %w", err)
		}
		name := filepath.Clean(filepath.FromSlash(hdr.Name))
		if name == "." || name == "" {
			continue
		}
		target := filepath.Join(destAbs, name)
		rel, err := filepath.Rel(destAbs, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("非法 tar 路径: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode)
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			// 跳过符号链接等，避免跨平台问题
			continue
		}
	}
	return nil
}
