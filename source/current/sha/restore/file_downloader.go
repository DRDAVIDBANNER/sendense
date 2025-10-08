// Package restore provides file download capabilities from mounted QCOW2 backups
// Task 4: File-Level Restore (Phase 3 - File Download & Extraction)
// Supports individual file downloads and directory archives (ZIP/TAR)
package restore

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// FileDownloader handles file and directory downloads from mounted backups
type FileDownloader struct {
	fileBrowser *FileBrowser
}

// NewFileDownloader creates a new file downloader instance
func NewFileDownloader(fileBrowser *FileBrowser) *FileDownloader {
	return &FileDownloader{
		fileBrowser: fileBrowser,
	}
}

// FileDownloadRequest represents a request to download a file
type FileDownloadRequest struct {
	MountID  string `json:"mount_id"`  // Required: Mount identifier
	FilePath string `json:"file_path"` // Required: Path to file within backup
}

// DirectoryDownloadRequest represents a request to download a directory
type DirectoryDownloadRequest struct {
	MountID     string `json:"mount_id"`      // Required: Mount identifier
	DirPath     string `json:"dir_path"`      // Required: Directory path within backup
	ArchiveType string `json:"archive_type"`  // "zip" or "tar.gz" (default: "zip")
}

// DownloadInfo contains metadata about a download operation
type DownloadInfo struct {
	FileName    string    `json:"file_name"`
	FilePath    string    `json:"file_path"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	StartedAt   time.Time `json:"started_at"`
}

// DownloadFile prepares a file for download
func (fd *FileDownloader) DownloadFile(ctx context.Context, req *FileDownloadRequest) (io.ReadCloser, *DownloadInfo, error) {
	log.WithFields(log.Fields{
		"mount_id":  req.MountID,
		"file_path": req.FilePath,
	}).Info("📥 Preparing file for download")

	// Validate file access (security check)
	safePath, err := fd.fileBrowser.ValidateFileAccess(ctx, req.MountID, req.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("file access validation failed: %w", err)
	}

	// Open file for reading
	file, err := os.Open(safePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Get file info
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Detect content type
	contentType := detectContentType(filepath.Ext(req.FilePath))

	downloadInfo := &DownloadInfo{
		FileName:    filepath.Base(req.FilePath),
		FilePath:    req.FilePath,
		Size:        info.Size(),
		ContentType: contentType,
		StartedAt:   time.Now(),
	}

	log.WithFields(log.Fields{
		"file_name":    downloadInfo.FileName,
		"size":         downloadInfo.Size,
		"content_type": downloadInfo.ContentType,
	}).Info("✅ File ready for download")

	return file, downloadInfo, nil
}

// DownloadDirectory prepares a directory as an archive (ZIP or TAR.GZ)
func (fd *FileDownloader) DownloadDirectory(ctx context.Context, req *DirectoryDownloadRequest) (io.ReadCloser, *DownloadInfo, error) {
	log.WithFields(log.Fields{
		"mount_id":     req.MountID,
		"dir_path":     req.DirPath,
		"archive_type": req.ArchiveType,
	}).Info("📦 Preparing directory archive for download")

	// Default to ZIP if not specified
	if req.ArchiveType == "" {
		req.ArchiveType = "zip"
	}

	// Validate archive type
	if req.ArchiveType != "zip" && req.ArchiveType != "tar.gz" {
		return nil, nil, fmt.Errorf("unsupported archive type: %s (supported: zip, tar.gz)", req.ArchiveType)
	}

	// Validate directory access (security check)
	safePath, err := fd.fileBrowser.ValidateDirectoryAccess(ctx, req.MountID, req.DirPath)
	if err != nil {
		return nil, nil, fmt.Errorf("directory access validation failed: %w", err)
	}

	// Calculate directory size (for progress tracking)
	totalSize, fileCount, err := fd.fileBrowser.CalculateDirectorySize(ctx, safePath)
	if err != nil {
		log.WithError(err).Warn("Failed to calculate directory size - continuing")
		totalSize = 0
	}

	log.WithFields(log.Fields{
		"dir_path":   req.DirPath,
		"total_size": totalSize,
		"file_count": fileCount,
	}).Info("📊 Directory size calculated")

	// Create archive
	reader, archiveSize, err := fd.createArchive(req.ArchiveType, safePath, req.DirPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create archive: %w", err)
	}

	// Determine content type
	contentType := "application/zip"
	if req.ArchiveType == "tar.gz" {
		contentType = "application/gzip"
	}

	// Generate archive filename
	archiveFileName := filepath.Base(req.DirPath) + "." + req.ArchiveType

	downloadInfo := &DownloadInfo{
		FileName:    archiveFileName,
		FilePath:    req.DirPath,
		Size:        archiveSize,
		ContentType: contentType,
		StartedAt:   time.Now(),
	}

	log.WithFields(log.Fields{
		"archive_name": archiveFileName,
		"archive_type": req.ArchiveType,
		"size":         archiveSize,
	}).Info("✅ Directory archive ready for download")

	return reader, downloadInfo, nil
}

// createArchive creates a ZIP or TAR.GZ archive from a directory
func (fd *FileDownloader) createArchive(archiveType, sourcePath, basePath string) (io.ReadCloser, int64, error) {
	log.WithFields(log.Fields{
		"archive_type": archiveType,
		"source_path":  sourcePath,
	}).Debug("🗜️  Creating archive")

	// Create pipe for streaming
	reader, writer := io.Pipe()

	// Start archive creation in goroutine
	go func() {
		var err error
		defer func() {
			if err != nil {
				writer.CloseWithError(err)
			} else {
				writer.Close()
			}
		}()

		if archiveType == "zip" {
			err = fd.createZIPArchive(writer, sourcePath, basePath)
		} else if archiveType == "tar.gz" {
			err = fd.createTarGzArchive(writer, sourcePath, basePath)
		}
	}()

	// Note: Size estimation for streaming archives
	// We return 0 for size as we're streaming - actual size determined during transfer
	return reader, 0, nil
}

// createZIPArchive creates a ZIP archive and streams it to the writer
func (fd *FileDownloader) createZIPArchive(w io.Writer, sourcePath, basePath string) error {
	log.WithField("source_path", sourcePath).Debug("📦 Creating ZIP archive")

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Walk directory and add files to ZIP
	err := filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.WithError(err).WithField("path", path).Warn("Error walking directory")
			return nil // Continue despite errors
		}

		// Skip directories (they're implicitly created)
		if d.IsDir() {
			return nil
		}

		// Get relative path for archive
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			log.WithError(err).WithField("path", path).Warn("Failed to get file info")
			return nil
		}

		// Create ZIP file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Use relative path as name in archive
		header.Name = relPath
		header.Method = zip.Deflate // Compression

		// Create entry in ZIP
		entryWriter, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Open source file
		file, err := os.Open(path)
		if err != nil {
			log.WithError(err).WithField("path", path).Warn("Failed to open file")
			return nil
		}
		defer file.Close()

		// Copy file contents to ZIP
		_, err = io.Copy(entryWriter, file)
		if err != nil {
			log.WithError(err).WithField("path", path).Warn("Failed to copy file to ZIP")
			return nil
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create ZIP archive: %w", err)
	}

	log.Info("✅ ZIP archive created")
	return nil
}

// createTarGzArchive creates a TAR.GZ archive and streams it to the writer
func (fd *FileDownloader) createTarGzArchive(w io.Writer, sourcePath, basePath string) error {
	log.WithField("source_path", sourcePath).Debug("📦 Creating TAR.GZ archive")

	// Create gzip writer
	gzipWriter := gzip.NewWriter(w)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk directory and add files to TAR
	err := filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.WithError(err).WithField("path", path).Warn("Error walking directory")
			return nil // Continue despite errors
		}

		// Get relative path for archive
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			log.WithError(err).WithField("path", path).Warn("Failed to get file info")
			return nil
		}

		// Create TAR header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Use relative path as name in archive
		header.Name = relPath

		// Write header to TAR
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If it's a file, copy contents
		if !d.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				log.WithError(err).WithField("path", path).Warn("Failed to open file")
				return nil
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				log.WithError(err).WithField("path", path).Warn("Failed to copy file to TAR")
				return nil
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create TAR.GZ archive: %w", err)
	}

	log.Info("✅ TAR.GZ archive created")
	return nil
}

// detectContentType determines MIME type based on file extension
func detectContentType(ext string) string {
	ext = strings.ToLower(ext)

	// Common content types
	contentTypes := map[string]string{
		".txt":  "text/plain",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".mp3":  "audio/mpeg",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	}

	if contentType, exists := contentTypes[ext]; exists {
		return contentType
	}

	// Default to octet-stream for unknown types
	return "application/octet-stream"
}

