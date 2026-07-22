package commercial

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/config"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

const MaxClientPackageSize int64 = 1 << 30

var allowedClientPackageSuffixes = []string{
	".tar.gz", ".appimage", ".exe", ".msi", ".zip", ".7z", ".dmg", ".pkg",
	".apk", ".ipa", ".deb", ".rpm", ".tar.xz", ".tar.zst",
}

func clientPackageDir() string {
	return filepath.Join(config.GetDBFolderPath(), "uploads", "client-applications")
}

func applicationDownloadURL(id string) string {
	return "/api/v1/user/applications/" + url.PathEscape(id) + "/download"
}

func populateApplicationDownload(row *model.ClientApplication) {
	if row != nil && strings.TrimSpace(row.PackageStoredName) != "" {
		row.DownloadURL = applicationDownloadURL(row.ID)
	}
}

func populateApplicationDownloads(rows []model.ClientApplication) {
	for index := range rows {
		populateApplicationDownload(&rows[index])
	}
}

func normalizedClientPackageName(raw string) (string, string, error) {
	name := filepath.Base(strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/"))
	if name == "" || name == "." || name == ".." || len([]byte(name)) > 240 {
		return "", "", errors.New("安装包文件名无效")
	}
	lower := strings.ToLower(name)
	for _, suffix := range allowedClientPackageSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return name, suffix, nil
		}
	}
	return "", "", errors.New("仅支持 EXE、MSI、ZIP、7Z、DMG、PKG、APK、IPA、AppImage、DEB、RPM 和 TAR 安装包")
}

func storedClientPackagePath(storedName string) (string, error) {
	storedName = strings.TrimSpace(storedName)
	if storedName == "" || filepath.Base(storedName) != storedName || strings.ContainsAny(storedName, `/\\`) {
		return "", errors.New("安装包存储路径无效")
	}
	return filepath.Join(clientPackageDir(), storedName), nil
}

func (s *AdminService) SaveApplicationPackage(id string, source io.Reader, rawFileName, contentType string) (*model.ClientApplication, error) {
	if source == nil {
		return nil, errors.New("请选择要上传的安装包")
	}

	var row model.ClientApplication
	if err := s.db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, errors.New("客户端入口不存在")
	}
	fileName, suffix, err := normalizedClientPackageName(rawFileName)
	if err != nil {
		return nil, err
	}

	dir := clientPackageDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("创建安装包目录失败: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return nil, fmt.Errorf("创建安装包临时文件失败: %w", err)
	}
	tempPath := temp.Name()
	keepTemp := false
	defer func() {
		_ = temp.Close()
		if !keepTemp {
			_ = os.Remove(tempPath)
		}
	}()

	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(temp, hash), io.LimitReader(source, MaxClientPackageSize+1))
	if copyErr != nil {
		return nil, fmt.Errorf("保存安装包失败: %w", copyErr)
	}
	if written <= 0 {
		return nil, errors.New("安装包不能为空")
	}
	if written > MaxClientPackageSize {
		return nil, errors.New("安装包不能超过 1 GB")
	}
	if err := temp.Sync(); err != nil {
		return nil, fmt.Errorf("同步安装包失败: %w", err)
	}
	if err := temp.Close(); err != nil {
		return nil, fmt.Errorf("关闭安装包失败: %w", err)
	}

	storedName := uuid.NewString() + suffix
	storedPath := filepath.Join(dir, storedName)
	if err := os.Rename(tempPath, storedPath); err != nil {
		return nil, fmt.Errorf("写入安装包失败: %w", err)
	}
	keepTemp = true
	if err := os.Chmod(storedPath, 0o640); err != nil {
		_ = os.Remove(storedPath)
		return nil, fmt.Errorf("设置安装包权限失败: %w", err)
	}

	oldStoredName := row.PackageStoredName
	updates := map[string]any{
		"package_file_name":    fileName,
		"package_stored_name":  storedName,
		"package_size":         written,
		"package_sha256":       hex.EncodeToString(hash.Sum(nil)),
		"package_content_type": strings.TrimSpace(contentType),
	}
	result := s.db.Model(&model.ClientApplication{}).Where("id = ?", row.ID).Updates(updates)
	if result.Error != nil || result.RowsAffected == 0 {
		_ = os.Remove(storedPath)
		if result.Error != nil {
			return nil, result.Error
		}
		return nil, errors.New("客户端入口不存在")
	}
	if oldPath, pathErr := storedClientPackagePath(oldStoredName); pathErr == nil && oldStoredName != storedName {
		_ = os.Remove(oldPath)
	}
	if err := s.db.First(&row, "id = ?", row.ID).Error; err != nil {
		return nil, err
	}
	populateApplicationDownload(&row)
	return &row, nil
}

func (s *PortalService) OpenApplicationPackage(id string) (*model.ClientApplication, *os.File, os.FileInfo, error) {
	var row model.ClientApplication
	err := s.db.Where("id = ? AND active = ? AND package_stored_name <> ''", strings.TrimSpace(id), true).First(&row).Error
	if err != nil {
		return nil, nil, nil, errors.New("安装包不存在或尚未开放下载")
	}
	path, err := storedClientPackagePath(row.PackageStoredName)
	if err != nil {
		return nil, nil, nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, errors.New("安装包文件不可用")
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, nil, nil, errors.New("安装包文件不可用")
	}
	return &row, file, info, nil
}
