package handler

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"ops-system/backend/internal/service"
	"ops-system/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	maxImportUploadBytes = 20 << 20 // 上传文件上限 20MB
	maxImportFileBytes   = 5 << 20  // 单个解压后文件上限 5MB（防解压炸弹）
	maxImportFileCount   = 200      // 压缩包内 YAML 文件数量上限
)

// ImportRules POST /api/v1/alerts/rules/import
// multipart form：file=<.yaml|.yml|.zip|.tar.gz|.tgz>，tenant_id=<uuid>。
func (h *AlertHandler) ImportRules(c *gin.Context) {
	tenantID, err := uuid.Parse(c.PostForm("tenant_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "invalid tenant_id")
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "file required")
		return
	}
	if fh.Size > maxImportUploadBytes {
		response.Error(c, http.StatusRequestEntityTooLarge, http.StatusRequestEntityTooLarge, response.ErrCodeValidation, "file too large (max 20MB)")
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, "cannot open uploaded file")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxImportUploadBytes+1))
	if err != nil || int64(len(data)) > maxImportUploadBytes {
		response.Error(c, http.StatusRequestEntityTooLarge, http.StatusRequestEntityTooLarge, response.ErrCodeValidation, "file too large (max 20MB)")
		return
	}

	files, err := extractRuleFiles(fh.Filename, data)
	if err != nil {
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
		return
	}

	result, err := h.alertSvc.ImportRules(c.Request.Context(), tenantID, files)
	if err != nil {
		if errors.Is(err, service.ErrNoRuleFiles) {
			response.Error(c, http.StatusBadRequest, http.StatusBadRequest, response.ErrCodeValidation, err.Error())
			return
		}
		h.handleErr(c, err)
		return
	}
	response.JSON(c, result)
}

// extractRuleFiles 根据文件名后缀提取 YAML 规则文件列表。
func extractRuleFiles(filename string, data []byte) ([]service.ImportRuleFile, error) {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"):
		return []service.ImportRuleFile{{Name: filename, Content: data}}, nil
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(data)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(data)
	default:
		return nil, fmt.Errorf("unsupported file type (accept .yaml/.yml/.zip/.tar.gz)")
	}
}

func isYamlEntry(name string) bool {
	base := path.Base(name)
	if strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") {
		return false
	}
	lower := strings.ToLower(base)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

func extractZip(data []byte) ([]service.ImportRuleFile, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip archive: %w", err)
	}
	var files []service.ImportRuleFile
	for _, entry := range zr.File {
		if entry.FileInfo().IsDir() || !isYamlEntry(entry.Name) {
			continue
		}
		if len(files) >= maxImportFileCount {
			return nil, fmt.Errorf("too many yaml files in archive (max %d)", maxImportFileCount)
		}
		rc, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", entry.Name, err)
		}
		content, err := io.ReadAll(io.LimitReader(rc, maxImportFileBytes+1))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", entry.Name, err)
		}
		if len(content) > maxImportFileBytes {
			return nil, fmt.Errorf("%s exceeds size limit (max 5MB)", entry.Name)
		}
		files = append(files, service.ImportRuleFile{Name: entry.Name, Content: content})
	}
	return files, nil
}

func extractTarGz(data []byte) ([]service.ImportRuleFile, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid gzip archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var files []service.ImportRuleFile
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid tar archive: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg || !isYamlEntry(hdr.Name) {
			continue
		}
		if len(files) >= maxImportFileCount {
			return nil, fmt.Errorf("too many yaml files in archive (max %d)", maxImportFileCount)
		}
		content, err := io.ReadAll(io.LimitReader(tr, maxImportFileBytes+1))
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", hdr.Name, err)
		}
		if len(content) > maxImportFileBytes {
			return nil, fmt.Errorf("%s exceeds size limit (max 5MB)", hdr.Name)
		}
		files = append(files, service.ImportRuleFile{Name: hdr.Name, Content: content})
	}
	return files, nil
}
