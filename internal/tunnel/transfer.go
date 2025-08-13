package tunnel

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
)

// FileTransfer provides advanced file transfer capabilities
type FileTransfer struct {
	client SSHClient
	tracer SSHTracer
	config *TransferConfig
	mu     sync.RWMutex
}

// TransferConfig holds advanced transfer configuration
type TransferConfig struct {
	ChunkSize            int64
	ProgressInterval     time.Duration
	ChecksumVerification bool
	ChecksumAlgorithm    ChecksumAlgorithm
	RetryAttempts        int
	RetryDelay           time.Duration
	AtomicOperations     bool
	PreservePermissions  bool
	PreserveTimestamps   bool
	MaxConcurrentOps     int
}

// DefaultTransferConfig returns default transfer configuration
func DefaultTransferConfig() *TransferConfig {
	return &TransferConfig{
		ChunkSize:            1024 * 1024, // 1MB chunks
		ProgressInterval:     100 * time.Millisecond,
		ChecksumVerification: true,
		ChecksumAlgorithm:    ChecksumSHA256,
		RetryAttempts:        3,
		RetryDelay:           time.Second,
		AtomicOperations:     true,
		PreservePermissions:  true,
		PreserveTimestamps:   true,
		MaxConcurrentOps:     5,
	}
}

// DefaultTransferOptions returns default transfer options
func DefaultTransferOptions() *TransferOptions {
	return &TransferOptions{
		Permissions:         0644,
		PreservePermissions: true,
		PreserveTimestamps:  true,
		VerifyChecksum:      true,
		ChecksumAlgorithm:   ChecksumSHA256,
		ChunkSize:           1024 * 1024, // 1MB
		AtomicOperation:     true,
		CreateDirectories:   true,
		OverwriteExisting:   false,
	}
}

// DefaultSyncOptions returns default sync options
func DefaultSyncOptions() *SyncOptions {
	return &SyncOptions{
		DeleteExtra:     false,
		PreserveLinks:   true,
		FollowLinks:     false,
		DryRun:          false,
		Concurrency:     3,
		VerifyIntegrity: true,
	}
}

// NewFileTransfer creates a new file transfer instance
func NewFileTransfer(client SSHClient, tracer SSHTracer, config *TransferConfig) *FileTransfer {
	if config == nil {
		config = DefaultTransferConfig()
	}

	return &FileTransfer{
		client: client,
		tracer: tracer,
		config: config,
	}
}

// UploadFile uploads a file to the remote server with advanced options
func (ft *FileTransfer) UploadFile(ctx context.Context, localPath, remotePath string, opts *TransferOptions) error {
	span := ft.tracer.TraceCommand(ctx, "upload_file", false)
	defer span.End()

	span.SetFields(map[string]any{
		"local_path":  localPath,
		"remote_path": remotePath,
		"atomic":      opts.AtomicOperation,
		"verify":      opts.VerifyChecksum,
	})

	if opts == nil {
		opts = DefaultTransferOptions()
	}

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	// Get file info
	localInfo, err := localFile.Stat()
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	// Create SFTP client
	sftpClient, err := ft.createSFTPClient(ctx)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Create remote directories if needed
	if opts.CreateDirectories {
		remoteDir := filepath.Dir(remotePath)
		if err := ft.createRemoteDirectories(sftpClient, remoteDir); err != nil {
			span.EndWithError(err)
			return fmt.Errorf("failed to create remote directories: %w", err)
		}
	}

	// Determine target path (atomic operation uses temp file)
	targetPath := remotePath
	if opts.AtomicOperation {
		targetPath = remotePath + ".tmp." + fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Check if file exists and handle overwrite
	if !opts.OverwriteExisting {
		if _, err := sftpClient.Stat(remotePath); err == nil {
			return fmt.Errorf("remote file already exists: %s", remotePath)
		}
	}

	// Upload file with progress tracking
	err = ft.uploadWithProgress(ctx, localFile, sftpClient, targetPath, localInfo.Size(), opts)
	if err != nil {
		// Clean up temp file on error
		if opts.AtomicOperation && targetPath != remotePath {
			sftpClient.Remove(targetPath)
		}
		span.EndWithError(err)
		return fmt.Errorf("failed to upload file: %w", err)
	}

	// Set permissions and ownership
	if err := ft.setRemoteFileAttributes(sftpClient, targetPath, localInfo, opts); err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to set file attributes: %w", err)
	}

	// Verify checksum if enabled
	if opts.VerifyChecksum {
		if err := ft.verifyUploadChecksum(ctx, localFile, sftpClient, targetPath, opts.ChecksumAlgorithm); err != nil {
			// Clean up on verification failure
			sftpClient.Remove(targetPath)
			span.EndWithError(err)
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// Atomic move if using temporary file
	if opts.AtomicOperation && targetPath != remotePath {
		if err := ft.atomicMove(sftpClient, targetPath, remotePath); err != nil {
			sftpClient.Remove(targetPath)
			span.EndWithError(err)
			return fmt.Errorf("atomic move failed: %w", err)
		}
	}

	span.Event("upload_completed")
	return nil
}

// DownloadFile downloads a file from the remote server with advanced options
func (ft *FileTransfer) DownloadFile(ctx context.Context, remotePath, localPath string, opts *TransferOptions) error {
	span := ft.tracer.TraceCommand(ctx, "download_file", false)
	defer span.End()

	span.SetFields(map[string]any{
		"remote_path": remotePath,
		"local_path":  localPath,
		"atomic":      opts.AtomicOperation,
		"verify":      opts.VerifyChecksum,
	})

	if opts == nil {
		opts = DefaultTransferOptions()
	}

	// Create SFTP client
	sftpClient, err := ft.createSFTPClient(ctx)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Get remote file info
	remoteInfo, err := sftpClient.Stat(remotePath)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to stat remote file: %w", err)
	}

	// Create local directories if needed
	if opts.CreateDirectories {
		localDir := filepath.Dir(localPath)
		if err := os.MkdirAll(localDir, 0755); err != nil {
			span.EndWithError(err)
			return fmt.Errorf("failed to create local directories: %w", err)
		}
	}

	// Determine target path (atomic operation uses temp file)
	targetPath := localPath
	if opts.AtomicOperation {
		targetPath = localPath + ".tmp." + fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Check if local file exists and handle overwrite
	if !opts.OverwriteExisting {
		if _, err := os.Stat(localPath); err == nil {
			return fmt.Errorf("local file already exists: %s", localPath)
		}
	}

	// Download file with progress tracking
	err = ft.downloadWithProgress(ctx, sftpClient, remotePath, targetPath, remoteInfo.Size(), opts)
	if err != nil {
		// Clean up temp file on error
		if opts.AtomicOperation && targetPath != localPath {
			os.Remove(targetPath)
		}
		span.EndWithError(err)
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Set local file attributes
	if err := ft.setLocalFileAttributes(targetPath, remoteInfo, opts); err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to set file attributes: %w", err)
	}

	// Verify checksum if enabled
	if opts.VerifyChecksum {
		if err := ft.verifyDownloadChecksum(ctx, sftpClient, remotePath, targetPath, opts.ChecksumAlgorithm); err != nil {
			// Clean up on verification failure
			os.Remove(targetPath)
			span.EndWithError(err)
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// Atomic move if using temporary file
	if opts.AtomicOperation && targetPath != localPath {
		if err := os.Rename(targetPath, localPath); err != nil {
			os.Remove(targetPath)
			span.EndWithError(err)
			return fmt.Errorf("atomic move failed: %w", err)
		}
	}

	span.Event("download_completed")
	return nil
}

// SyncDirectory synchronizes directories between local and remote
func (ft *FileTransfer) SyncDirectory(ctx context.Context, sourcePath, destPath string, direction TransferDirection, opts *SyncOptions) (*SyncResult, error) {
	span := ft.tracer.TraceCommand(ctx, "sync_directory", false)
	defer span.End()

	span.SetFields(map[string]any{
		"source_path": sourcePath,
		"dest_path":   destPath,
		"direction":   int(direction),
		"dry_run":     opts.DryRun,
	})

	if opts == nil {
		opts = DefaultSyncOptions()
	}

	result := &SyncResult{
		StartTime: time.Now(),
	}

	// Create SFTP client
	sftpClient, err := ft.createSFTPClient(ctx)
	if err != nil {
		span.EndWithError(err)
		return result, fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Perform synchronization based on direction
	switch direction {
	case TransferUpload:
		err = ft.syncLocalToRemote(ctx, sourcePath, destPath, sftpClient, opts, result)
	case TransferDownload:
		err = ft.syncRemoteToLocal(ctx, sourcePath, destPath, sftpClient, opts, result)
	default:
		err = fmt.Errorf("unsupported sync direction: %d", direction)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		span.EndWithError(err)
		return result, err
	}

	span.Event("sync_completed")
	return result, nil
}

// CreateRemoteFile creates a file on the remote server with content
func (ft *FileTransfer) CreateRemoteFile(ctx context.Context, remotePath string, content []byte, perms os.FileMode) error {
	span := ft.tracer.TraceCommand(ctx, "create_remote_file", false)
	defer span.End()

	span.SetFields(map[string]any{
		"remote_path": remotePath,
		"size":        len(content),
		"permissions": perms,
	})

	// Create SFTP client
	sftpClient, err := ft.createSFTPClient(ctx)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Create remote directories if needed
	remoteDir := filepath.Dir(remotePath)
	if err := ft.createRemoteDirectories(sftpClient, remoteDir); err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to create remote directories: %w", err)
	}

	// Determine target path for atomic operation
	targetPath := remotePath
	if ft.config.AtomicOperations {
		targetPath = remotePath + ".tmp." + fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Create remote file
	remoteFile, err := sftpClient.Create(targetPath)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer remoteFile.Close()

	// Write content
	_, err = remoteFile.Write(content)
	if err != nil {
		sftpClient.Remove(targetPath)
		span.EndWithError(err)
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Set permissions
	if err := sftpClient.Chmod(targetPath, perms); err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic move if using temporary file
	if ft.config.AtomicOperations && targetPath != remotePath {
		if err := ft.atomicMove(sftpClient, targetPath, remotePath); err != nil {
			sftpClient.Remove(targetPath)
			span.EndWithError(err)
			return fmt.Errorf("atomic move failed: %w", err)
		}
	}

	span.Event("file_created")
	return nil
}

// GetRemoteFileInfo retrieves information about a remote file
func (ft *FileTransfer) GetRemoteFileInfo(ctx context.Context, remotePath string) (os.FileInfo, error) {
	span := ft.tracer.TraceCommand(ctx, "get_remote_file_info", false)
	defer span.End()

	span.SetFields(map[string]any{
		"remote_path": remotePath,
	})

	// Create SFTP client
	sftpClient, err := ft.createSFTPClient(ctx)
	if err != nil {
		span.EndWithError(err)
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Get file info
	info, err := sftpClient.Stat(remotePath)
	if err != nil {
		span.EndWithError(err)
		return nil, fmt.Errorf("failed to stat remote file: %w", err)
	}

	span.Event("file_info_retrieved")
	return info, nil
}

// RemoveRemoteFile removes a file from the remote server
func (ft *FileTransfer) RemoveRemoteFile(ctx context.Context, remotePath string) error {
	span := ft.tracer.TraceCommand(ctx, "remove_remote_file", false)
	defer span.End()

	span.SetFields(map[string]any{
		"remote_path": remotePath,
	})

	// Create SFTP client
	sftpClient, err := ft.createSFTPClient(ctx)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Remove file
	err = sftpClient.Remove(remotePath)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to remove remote file: %w", err)
	}

	span.Event("file_removed")
	return nil
}

// createSFTPClient creates an SFTP client from the SSH connection
func (ft *FileTransfer) createSFTPClient(ctx context.Context) (*sftp.Client, error) {
	// Get underlying SSH connection
	sshClient, ok := ft.client.(*sshClient)
	if !ok {
		return nil, fmt.Errorf("unsupported client type for SFTP operations")
	}

	sshClient.mu.RLock()
	conn := sshClient.conn
	sshClient.mu.RUnlock()

	if conn == nil {
		return nil, ErrClientNotConnected
	}

	// Create SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	return client, nil
}

// uploadWithProgress uploads a file with progress tracking
func (ft *FileTransfer) uploadWithProgress(ctx context.Context, localFile *os.File, sftpClient *sftp.Client, remotePath string, totalSize int64, opts *TransferOptions) error {
	// Create remote file
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer remoteFile.Close()

	// Track progress
	var transferred int64
	var lastUpdate time.Time

	// Create progress tracker
	progressReader := &progressReader{
		reader: localFile,
		callback: func(n int64) {
			transferred += n
			now := time.Now()

			// Update progress at configured intervals
			if now.Sub(lastUpdate) >= ft.config.ProgressInterval {
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(transferred, totalSize)
				}
				lastUpdate = now
			}
		},
	}

	// Copy with chunked reading
	buffer := make([]byte, opts.ChunkSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := progressReader.Read(buffer)
		if n > 0 {
			_, writeErr := remoteFile.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write to remote file: %w", writeErr)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read local file: %w", err)
		}
	}

	// Final progress update
	if opts.ProgressCallback != nil {
		opts.ProgressCallback(transferred, totalSize)
	}

	return nil
}

// downloadWithProgress downloads a file with progress tracking
func (ft *FileTransfer) downloadWithProgress(ctx context.Context, sftpClient *sftp.Client, remotePath, localPath string, totalSize int64, opts *TransferOptions) error {
	// Open remote file
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Track progress
	var transferred int64
	var lastUpdate time.Time

	// Create progress tracker
	progressWriter := &progressWriter{
		writer: localFile,
		callback: func(n int64) {
			transferred += n
			now := time.Now()

			// Update progress at configured intervals
			if now.Sub(lastUpdate) >= ft.config.ProgressInterval {
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(transferred, totalSize)
				}
				lastUpdate = now
			}
		},
	}

	// Copy with chunked reading
	buffer := make([]byte, opts.ChunkSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := remoteFile.Read(buffer)
		if n > 0 {
			_, writeErr := progressWriter.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write to local file: %w", writeErr)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read remote file: %w", err)
		}
	}

	// Final progress update
	if opts.ProgressCallback != nil {
		opts.ProgressCallback(transferred, totalSize)
	}

	return nil
}

// syncLocalToRemote synchronizes local directory to remote
func (ft *FileTransfer) syncLocalToRemote(ctx context.Context, localPath, remotePath string, sftpClient *sftp.Client, opts *SyncOptions, result *SyncResult) error {
	// Walk local directory
	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, err)
			return nil // Continue walking
		}

		// Calculate relative path
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			result.Errors = append(result.Errors, err)
			return nil
		}

		remoteDest := filepath.Join(remotePath, relPath)

		// Handle directories
		if info.IsDir() {
			if !opts.DryRun {
				if err := ft.createRemoteDirectories(sftpClient, remoteDest); err != nil {
					result.Errors = append(result.Errors, err)
				}
			}
			return nil
		}

		// Skip if doesn't match include patterns
		if !ft.matchesPatterns(relPath, opts.IncludePatterns, opts.ExcludePatterns) {
			result.FilesSkipped++
			return nil
		}

		// Check if file needs to be transferred
		needsTransfer, err := ft.needsTransfer(sftpClient, path, remoteDest, info)
		if err != nil {
			result.Errors = append(result.Errors, err)
			return nil
		}

		if !needsTransfer {
			result.FilesSkipped++
			return nil
		}

		// Transfer file
		if !opts.DryRun {
			transferOpts := DefaultTransferOptions()
			transferOpts.PreservePermissions = true
			transferOpts.PreserveTimestamps = true
			transferOpts.CreateDirectories = true
			transferOpts.OverwriteExisting = true

			if opts.ProgressCallback != nil {
				transferOpts.ProgressCallback = func(transferred, total int64) {
					opts.ProgressCallback("upload", relPath, transferred, total)
				}
			}

			if err := ft.uploadFileInternal(ctx, path, remoteDest, sftpClient, transferOpts); err != nil {
				result.Errors = append(result.Errors, err)
				return nil
			}
		}

		result.FilesTransferred++
		result.BytesTransferred += info.Size()

		return nil
	})
}

// syncRemoteToLocal synchronizes remote directory to local
func (ft *FileTransfer) syncRemoteToLocal(ctx context.Context, remotePath, localPath string, sftpClient *sftp.Client, opts *SyncOptions, result *SyncResult) error {
	// Walk remote directory
	walker := sftpClient.Walk(remotePath)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		path := walker.Path()
		info := walker.Stat()

		// Calculate relative path
		relPath, err := filepath.Rel(remotePath, path)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		localDest := filepath.Join(localPath, relPath)

		// Handle directories
		if info.IsDir() {
			if !opts.DryRun {
				if err := os.MkdirAll(localDest, info.Mode()); err != nil {
					result.Errors = append(result.Errors, err)
				}
			}
			continue
		}

		// Skip if doesn't match include patterns
		if !ft.matchesPatterns(relPath, opts.IncludePatterns, opts.ExcludePatterns) {
			result.FilesSkipped++
			continue
		}

		// Check if file needs to be transferred
		needsTransfer, err := ft.needsTransferRemote(sftpClient, path, localDest, info)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		if !needsTransfer {
			result.FilesSkipped++
			continue
		}

		// Transfer file
		if !opts.DryRun {
			transferOpts := DefaultTransferOptions()
			transferOpts.PreservePermissions = true
			transferOpts.PreserveTimestamps = true
			transferOpts.CreateDirectories = true
			transferOpts.OverwriteExisting = true

			if opts.ProgressCallback != nil {
				transferOpts.ProgressCallback = func(transferred, total int64) {
					opts.ProgressCallback("download", relPath, transferred, total)
				}
			}

			if err := ft.downloadFileInternal(ctx, path, localDest, sftpClient, transferOpts); err != nil {
				result.Errors = append(result.Errors, err)
				continue
			}
		}

		result.FilesTransferred++
		result.BytesTransferred += info.Size()
	}

	return nil
}

// needsTransfer determines if a local file needs to be transferred to remote
func (ft *FileTransfer) needsTransfer(sftpClient *sftp.Client, localPath, remotePath string, localInfo os.FileInfo) (bool, error) {
	// Check if remote file exists
	remoteInfo, err := sftpClient.Stat(remotePath)
	if err != nil {
		// File doesn't exist, needs transfer
		return true, nil
	}

	// Compare modification times
	if localInfo.ModTime().After(remoteInfo.ModTime()) {
		return true, nil
	}

	// Compare file sizes
	if localInfo.Size() != remoteInfo.Size() {
		return true, nil
	}

	return false, nil
}

// needsTransferRemote determines if a remote file needs to be transferred to local
func (ft *FileTransfer) needsTransferRemote(sftpClient *sftp.Client, remotePath, localPath string, remoteInfo os.FileInfo) (bool, error) {
	// Check if local file exists
	localInfo, err := os.Stat(localPath)
	if err != nil {
		// File doesn't exist, needs transfer
		return true, nil
	}

	// Compare modification times
	if remoteInfo.ModTime().After(localInfo.ModTime()) {
		return true, nil
	}

	// Compare file sizes
	if remoteInfo.Size() != localInfo.Size() {
		return true, nil
	}

	return false, nil
}

// matchesPatterns checks if a path matches include/exclude patterns
func (ft *FileTransfer) matchesPatterns(path string, includePatterns, excludePatterns []string) bool {
	// Check exclude patterns first
	for _, pattern := range excludePatterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return false
		}
	}

	// If no include patterns, include by default
	if len(includePatterns) == 0 {
		return true
	}

	// Check include patterns
	for _, pattern := range includePatterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}

	return false
}

// createRemoteDirectories creates remote directories recursively
func (ft *FileTransfer) createRemoteDirectories(sftpClient *sftp.Client, remotePath string) error {
	// Normalize path
	remotePath = filepath.Clean(remotePath)
	if remotePath == "." || remotePath == "/" {
		return nil
	}

	// Check if directory already exists
	if _, err := sftpClient.Stat(remotePath); err == nil {
		return nil // Directory exists
	}

	// Create parent directories first
	parentDir := filepath.Dir(remotePath)
	if parentDir != remotePath {
		if err := ft.createRemoteDirectories(sftpClient, parentDir); err != nil {
			return err
		}
	}

	// Create this directory
	return sftpClient.Mkdir(remotePath)
}

// setRemoteFileAttributes sets permissions, ownership, and timestamps on remote file
func (ft *FileTransfer) setRemoteFileAttributes(sftpClient *sftp.Client, remotePath string, localInfo os.FileInfo, opts *TransferOptions) error {
	// Set permissions
	if opts.PreservePermissions {
		if err := sftpClient.Chmod(remotePath, localInfo.Mode()); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	} else if opts.Permissions != 0 {
		if err := sftpClient.Chmod(remotePath, opts.Permissions); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	// Set timestamps
	if opts.PreserveTimestamps {
		if err := sftpClient.Chtimes(remotePath, localInfo.ModTime(), localInfo.ModTime()); err != nil {
			return fmt.Errorf("failed to set timestamps: %w", err)
		}
	}

	// Set ownership if specified
	if opts.Owner != "" || opts.Group != "" {
		if err := ft.setRemoteOwnership(sftpClient, remotePath, opts.Owner, opts.Group); err != nil {
			return fmt.Errorf("failed to set ownership: %w", err)
		}
	}

	return nil
}

// setLocalFileAttributes sets permissions and timestamps on local file
func (ft *FileTransfer) setLocalFileAttributes(localPath string, remoteInfo os.FileInfo, opts *TransferOptions) error {
	// Set permissions
	if opts.PreservePermissions {
		if err := os.Chmod(localPath, remoteInfo.Mode()); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	} else if opts.Permissions != 0 {
		if err := os.Chmod(localPath, opts.Permissions); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	// Set timestamps
	if opts.PreserveTimestamps {
		if err := os.Chtimes(localPath, remoteInfo.ModTime(), remoteInfo.ModTime()); err != nil {
			return fmt.Errorf("failed to set timestamps: %w", err)
		}
	}

	return nil
}

// setRemoteOwnership sets ownership on remote file using chown command
func (ft *FileTransfer) setRemoteOwnership(sftpClient *sftp.Client, remotePath, owner, group string) error {
	// Build chown command
	chownTarget := ""
	if owner != "" && group != "" {
		chownTarget = fmt.Sprintf("%s:%s", owner, group)
	} else if owner != "" {
		chownTarget = owner
	} else if group != "" {
		chownTarget = fmt.Sprintf(":%s", group)
	} else {
		return nil // Nothing to do
	}

	// Execute chown via SSH
	cmdStr := fmt.Sprintf("chown %s %s", chownTarget, remotePath)
	if _, err := ft.client.Execute(context.Background(), cmdStr); err != nil {
		return fmt.Errorf("failed to change ownership: %w", err)
	}
	return nil
}

// verifyUploadChecksum verifies the checksum of an uploaded file
func (ft *FileTransfer) verifyUploadChecksum(ctx context.Context, localFile *os.File, sftpClient *sftp.Client, remotePath string, algorithm ChecksumAlgorithm) error {
	// Calculate local checksum
	localChecksum, err := ft.calculateLocalChecksum(localFile, algorithm)
	if err != nil {
		return fmt.Errorf("failed to calculate local checksum: %w", err)
	}

	// Calculate remote checksum
	remoteChecksum, err := ft.calculateRemoteChecksum(sftpClient, remotePath, algorithm)
	if err != nil {
		return fmt.Errorf("failed to calculate remote checksum: %w", err)
	}

	// Compare checksums
	if localChecksum != remoteChecksum {
		return fmt.Errorf("checksum mismatch: local=%s, remote=%s", localChecksum, remoteChecksum)
	}

	return nil
}

// verifyDownloadChecksum verifies the checksum of a downloaded file
func (ft *FileTransfer) verifyDownloadChecksum(ctx context.Context, sftpClient *sftp.Client, remotePath, localPath string, algorithm ChecksumAlgorithm) error {
	// Calculate remote checksum
	remoteChecksum, err := ft.calculateRemoteChecksum(sftpClient, remotePath, algorithm)
	if err != nil {
		return fmt.Errorf("failed to calculate remote checksum: %w", err)
	}

	// Calculate local checksum
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file for checksum: %w", err)
	}
	defer localFile.Close()

	localChecksum, err := ft.calculateLocalChecksum(localFile, algorithm)
	if err != nil {
		return fmt.Errorf("failed to calculate local checksum: %w", err)
	}

	// Compare checksums
	if localChecksum != remoteChecksum {
		return fmt.Errorf("checksum mismatch: local=%s, remote=%s", localChecksum, remoteChecksum)
	}

	return nil
}

// calculateLocalChecksum calculates checksum of a local file
func (ft *FileTransfer) calculateLocalChecksum(file *os.File, algorithm ChecksumAlgorithm) (string, error) {
	// Reset file position
	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}

	return ft.calculateChecksum(file, algorithm)
}

// calculateRemoteChecksum calculates checksum of a remote file via command execution
func (ft *FileTransfer) calculateRemoteChecksum(sftpClient *sftp.Client, remotePath string, algorithm ChecksumAlgorithm) (string, error) {
	var cmdStr string
	switch algorithm {
	case ChecksumMD5:
		cmdStr = fmt.Sprintf("md5sum %s | cut -d' ' -f1", remotePath)
	case ChecksumSHA256:
		cmdStr = fmt.Sprintf("sha256sum %s | cut -d' ' -f1", remotePath)
	default:
		return "", fmt.Errorf("unsupported checksum algorithm: %s", algorithm)
	}

	// Execute checksum command
	output, err := ft.client.Execute(context.Background(), cmdStr)
	if err != nil {
		return "", fmt.Errorf("failed to execute checksum command: %w", err)
	}

	checksum := strings.TrimSpace(output)
	if checksum == "" {
		return "", fmt.Errorf("empty checksum result")
	}

	return checksum, nil
}

// calculateChecksum calculates checksum of a reader
func (ft *FileTransfer) calculateChecksum(reader io.Reader, algorithm ChecksumAlgorithm) (string, error) {
	switch algorithm {
	case ChecksumMD5:
		return calculateMD5(reader)
	case ChecksumSHA256:
		return calculateSHA256(reader)
	default:
		return "", fmt.Errorf("unsupported checksum algorithm: %s", algorithm)
	}
}

// atomicMove performs an atomic move operation on remote file
func (ft *FileTransfer) atomicMove(sftpClient *sftp.Client, sourcePath, destPath string) error {
	// Use SFTP rename operation which should be atomic on most filesystems
	return sftpClient.Rename(sourcePath, destPath)
}

// uploadFileInternal uploads a file using existing SFTP client
func (ft *FileTransfer) uploadFileInternal(ctx context.Context, localPath, remotePath string, sftpClient *sftp.Client, opts *TransferOptions) error {
	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	// Get file info
	localInfo, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	// Upload with progress
	err = ft.uploadWithProgress(ctx, localFile, sftpClient, remotePath, localInfo.Size(), opts)
	if err != nil {
		return err
	}

	// Set attributes
	return ft.setRemoteFileAttributes(sftpClient, remotePath, localInfo, opts)
}

// downloadFileInternal downloads a file using existing SFTP client
func (ft *FileTransfer) downloadFileInternal(ctx context.Context, remotePath, localPath string, sftpClient *sftp.Client, opts *TransferOptions) error {
	// Get remote file info
	remoteInfo, err := sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("failed to stat remote file: %w", err)
	}

	// Download with progress
	err = ft.downloadWithProgress(ctx, sftpClient, remotePath, localPath, remoteInfo.Size(), opts)
	if err != nil {
		return err
	}

	// Set attributes
	return ft.setLocalFileAttributes(localPath, remoteInfo, opts)
}

// progressReader wraps a reader to track progress
type progressReader struct {
	reader   io.Reader
	callback func(int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 && pr.callback != nil {
		pr.callback(int64(n))
	}
	return n, err
}

// progressWriter wraps a writer to track progress
type progressWriter struct {
	writer   io.Writer
	callback func(int64)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if n > 0 && pw.callback != nil {
		pw.callback(int64(n))
	}
	return n, err
}

// calculateMD5 calculates MD5 checksum of a reader
func calculateMD5(reader io.Reader) (string, error) {
	hasher := md5.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// calculateSHA256 calculates SHA256 checksum of a reader
func calculateSHA256(reader io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// BatchTransfer performs multiple file transfers with optional concurrency
func (ft *FileTransfer) BatchTransfer(ctx context.Context, operations []BatchTransferOperation, maxConcurrency int) error {
	span := ft.tracer.TraceCommand(ctx, "batch_transfer", false)
	defer span.End()

	span.SetFields(map[string]any{
		"operation_count": len(operations),
		"max_concurrency": maxConcurrency,
	})

	if maxConcurrency <= 0 {
		maxConcurrency = ft.config.MaxConcurrentOps
	}

	// Create semaphore to limit concurrency
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	// Process operations
	for i, op := range operations {
		wg.Add(1)
		go func(index int, operation BatchTransferOperation) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			// Perform transfer
			var err error
			switch operation.Direction {
			case TransferUpload:
				err = ft.UploadFile(ctx, operation.LocalPath, operation.RemotePath, operation.Options)
			case TransferDownload:
				err = ft.DownloadFile(ctx, operation.RemotePath, operation.LocalPath, operation.Options)
			default:
				err = fmt.Errorf("unsupported direction for operation %d", index)
			}

			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("operation %d failed: %w", index, err))
				mu.Unlock()
			}
		}(i, op)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Return combined errors if any
	if len(errors) > 0 {
		var errMsg strings.Builder
		errMsg.WriteString("batch transfer failed with errors:\n")
		for _, err := range errors {
			errMsg.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
		}
		err := fmt.Errorf("%s", errMsg.String())
		span.EndWithError(err)
		return err
	}

	span.Event("batch_transfer_completed")
	return nil
}

// GetTransferProgress creates a progress callback that tracks transfer metrics
func (ft *FileTransfer) GetTransferProgress(fileName string) (*TransferProgress, func(int64, int64)) {
	progress := &TransferProgress{
		FileName:  fileName,
		StartTime: time.Now(),
	}

	callback := func(transferred, total int64) {
		now := time.Now()
		progress.BytesTotal = total
		progress.BytesTransferred = transferred

		if total > 0 {
			progress.Percentage = float64(transferred) / float64(total) * 100
		}

		// Calculate speed
		elapsed := now.Sub(progress.StartTime)
		if elapsed > 0 {
			progress.Speed = int64(float64(transferred) / elapsed.Seconds())
		}

		// Calculate ETA
		if progress.Speed > 0 && transferred < total {
			remaining := total - transferred
			progress.ETA = time.Duration(float64(remaining)/float64(progress.Speed)) * time.Second
		}

		progress.LastUpdate = now
	}

	return progress, callback
}

// ValidateTransferPath validates a transfer path for security
func (ft *FileTransfer) ValidateTransferPath(path string) error {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected in path: %s", path)
	}

	// Check for absolute paths in relative context
	if filepath.IsAbs(cleanPath) && !strings.HasPrefix(cleanPath, "/") {
		return fmt.Errorf("suspicious absolute path: %s", path)
	}

	// Check for dangerous characters
	dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "<", ">"}
	for _, char := range dangerousChars {
		if strings.Contains(path, char) {
			return fmt.Errorf("dangerous character '%s' in path: %s", char, path)
		}
	}

	return nil
}

// EstimateTransferTime estimates transfer time based on file size and connection speed
func (ft *FileTransfer) EstimateTransferTime(fileSize int64, connectionSpeedBps int64) time.Duration {
	if connectionSpeedBps <= 0 {
		return 0
	}

	// Add 20% overhead for protocol and processing
	overhead := float64(fileSize) * 0.2
	totalBytes := float64(fileSize) + overhead

	seconds := totalBytes / float64(connectionSpeedBps)
	return time.Duration(seconds * float64(time.Second))
}

// CleanupTempFiles removes temporary files created during failed operations
func (ft *FileTransfer) CleanupTempFiles(ctx context.Context, pattern string) error {
	span := ft.tracer.TraceCommand(ctx, "cleanup_temp_files", false)
	defer span.End()

	span.SetFields(map[string]any{
		"pattern": pattern,
	})

	// Create SFTP client
	sftpClient, err := ft.createSFTPClient(ctx)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Find and remove temporary files
	cmdStr := fmt.Sprintf("find /tmp -name '%s' -type f -mtime +1 -delete", pattern)
	if _, err := ft.client.Execute(ctx, cmdStr); err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to cleanup temp files: %w", err)
	}

	span.Event("cleanup_completed")
	return nil
}

// GetDiskSpace retrieves disk space information for a remote path
func (ft *FileTransfer) GetDiskSpace(ctx context.Context, remotePath string) (*DiskSpaceInfo, error) {
	span := ft.tracer.TraceCommand(ctx, "get_disk_space", false)
	defer span.End()

	span.SetFields(map[string]any{
		"remote_path": remotePath,
	})

	// Execute df command
	cmdStr := fmt.Sprintf("df -B1 %s | tail -1", remotePath)
	output, err := ft.client.Execute(ctx, cmdStr)
	if err != nil {
		span.EndWithError(err)
		return nil, fmt.Errorf("failed to get disk space: %w", err)
	}

	// Parse df output
	fields := strings.Fields(output)
	if len(fields) < 4 {
		return nil, fmt.Errorf("unexpected df output format")
	}

	diskInfo := &DiskSpaceInfo{
		Path: remotePath,
	}

	// Parse numeric fields (df output: filesystem, total, used, available, use%, mount)
	if len(fields) >= 6 {
		diskInfo.Filesystem = fields[0]
		diskInfo.Total = parseInt64(fields[1])
		diskInfo.Used = parseInt64(fields[2])
		diskInfo.Available = parseInt64(fields[3])
		diskInfo.MountPoint = fields[5]
	}

	span.Event("disk_space_retrieved")
	return diskInfo, nil
}

// parseInt64 safely parses a string to int64
func parseInt64(s string) int64 {
	result, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return result
}

// TransferStats provides statistics about file transfer operations
type TransferStats struct {
	TotalFiles       int64
	TotalBytes       int64
	TransferredFiles int64
	TransferredBytes int64
	FailedFiles      int64
	StartTime        time.Time
	EndTime          time.Time
	AverageSpeed     int64 // bytes per second
}

// NewTransferStats creates a new transfer statistics tracker
func NewTransferStats() *TransferStats {
	return &TransferStats{
		StartTime: time.Now(),
	}
}

// UpdateStats updates transfer statistics
func (ts *TransferStats) UpdateStats(fileSize int64, success bool) {
	ts.TotalFiles++
	ts.TotalBytes += fileSize

	if success {
		ts.TransferredFiles++
		ts.TransferredBytes += fileSize
	} else {
		ts.FailedFiles++
	}

	ts.EndTime = time.Now()
	duration := ts.EndTime.Sub(ts.StartTime)
	if duration > 0 {
		ts.AverageSpeed = int64(float64(ts.TransferredBytes) / duration.Seconds())
	}
}

// GetSuccessRate returns the success rate as a percentage
func (ts *TransferStats) GetSuccessRate() float64 {
	if ts.TotalFiles == 0 {
		return 0
	}
	return float64(ts.TransferredFiles) / float64(ts.TotalFiles) * 100
}

// TransferSession manages a transfer session with connection reuse
type TransferSession struct {
	ft         *FileTransfer
	sftpClient *sftp.Client
	stats      *TransferStats
	mu         sync.Mutex
}

// NewTransferSession creates a new transfer session
func (ft *FileTransfer) NewTransferSession(ctx context.Context) (*TransferSession, error) {
	sftpClient, err := ft.createSFTPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	return &TransferSession{
		ft:         ft,
		sftpClient: sftpClient,
		stats:      NewTransferStats(),
	}, nil
}

// Close closes the transfer session
func (ts *TransferSession) Close() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.sftpClient != nil {
		return ts.sftpClient.Close()
	}
	return nil
}

// UploadFile uploads a file within the session
func (ts *TransferSession) UploadFile(ctx context.Context, localPath, remotePath string, opts *TransferOptions) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if opts == nil {
		opts = DefaultTransferOptions()
	}

	// Get file size for stats
	localInfo, err := os.Stat(localPath)
	if err != nil {
		ts.stats.UpdateStats(0, false)
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	err = ts.ft.uploadFileInternal(ctx, localPath, remotePath, ts.sftpClient, opts)
	ts.stats.UpdateStats(localInfo.Size(), err == nil)
	return err
}

// DownloadFile downloads a file within the session
func (ts *TransferSession) DownloadFile(ctx context.Context, remotePath, localPath string, opts *TransferOptions) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if opts == nil {
		opts = DefaultTransferOptions()
	}

	// Get file size for stats
	remoteInfo, err := ts.sftpClient.Stat(remotePath)
	if err != nil {
		ts.stats.UpdateStats(0, false)
		return fmt.Errorf("failed to stat remote file: %w", err)
	}

	err = ts.ft.downloadFileInternal(ctx, remotePath, localPath, ts.sftpClient, opts)
	ts.stats.UpdateStats(remoteInfo.Size(), err == nil)
	return err
}

// GetStats returns current transfer statistics
func (ts *TransferSession) GetStats() *TransferStats {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Return a copy to prevent race conditions
	statsCopy := *ts.stats
	return &statsCopy
}

// ResumeTransfer resumes a partially transferred file
func (ft *FileTransfer) ResumeTransfer(ctx context.Context, localPath, remotePath string, direction TransferDirection, opts *TransferOptions) error {
	span := ft.tracer.TraceCommand(ctx, "resume_transfer", false)
	defer span.End()

	span.SetFields(map[string]any{
		"local_path":  localPath,
		"remote_path": remotePath,
		"direction":   int(direction),
	})

	if opts == nil {
		opts = DefaultTransferOptions()
	}

	// Create SFTP client
	sftpClient, err := ft.createSFTPClient(ctx)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	switch direction {
	case TransferUpload:
		return ft.resumeUpload(ctx, localPath, remotePath, sftpClient, opts)
	case TransferDownload:
		return ft.resumeDownload(ctx, remotePath, localPath, sftpClient, opts)
	default:
		return fmt.Errorf("unsupported transfer direction: %d", direction)
	}
}

// resumeUpload resumes an upload operation
func (ft *FileTransfer) resumeUpload(ctx context.Context, localPath, remotePath string, sftpClient *sftp.Client, opts *TransferOptions) error {
	// Get local file info
	localInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	// Check if remote file exists and get its size
	var remoteSize int64 = 0
	if remoteInfo, err := sftpClient.Stat(remotePath); err == nil {
		remoteSize = remoteInfo.Size()
	}

	// If remote file is same size or larger, assume complete
	if remoteSize >= localInfo.Size() {
		return nil
	}

	// Open local file and seek to resume position
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	if _, err := localFile.Seek(remoteSize, 0); err != nil {
		return fmt.Errorf("failed to seek in local file: %w", err)
	}

	// Open remote file in append mode
	remoteFile, err := sftpClient.OpenFile(remotePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE)
	if err != nil {
		return fmt.Errorf("failed to open remote file for append: %w", err)
	}
	defer remoteFile.Close()

	// Resume transfer with progress tracking
	progressReader := &progressReader{
		reader: localFile,
		callback: func(n int64) {
			if opts.ProgressCallback != nil {
				opts.ProgressCallback(remoteSize+n, localInfo.Size())
			}
		},
	}

	buffer := make([]byte, opts.ChunkSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := progressReader.Read(buffer)
		if n > 0 {
			_, writeErr := remoteFile.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write to remote file: %w", writeErr)
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read local file: %w", readErr)
		}
	}

	return nil
}

// resumeDownload resumes a download operation
func (ft *FileTransfer) resumeDownload(ctx context.Context, remotePath, localPath string, sftpClient *sftp.Client, opts *TransferOptions) error {
	// Get remote file info
	remoteInfo, err := sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("failed to stat remote file: %w", err)
	}

	// Check if local file exists and get its size
	var localSize int64 = 0
	if localInfo, err := os.Stat(localPath); err == nil {
		localSize = localInfo.Size()
	}

	// If local file is same size or larger, assume complete
	if localSize >= remoteInfo.Size() {
		return nil
	}

	// Open remote file and seek to resume position
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	if _, err := remoteFile.Seek(localSize, 0); err != nil {
		return fmt.Errorf("failed to seek in remote file: %w", err)
	}

	// Open local file in append mode
	localFile, err := os.OpenFile(localPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open local file for append: %w", err)
	}
	defer localFile.Close()

	// Resume transfer with progress tracking
	progressWriter := &progressWriter{
		writer: localFile,
		callback: func(n int64) {
			if opts.ProgressCallback != nil {
				opts.ProgressCallback(localSize+n, remoteInfo.Size())
			}
		},
	}

	buffer := make([]byte, opts.ChunkSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := remoteFile.Read(buffer)
		if n > 0 {
			_, writeErr := progressWriter.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write to local file: %w", writeErr)
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read remote file: %w", readErr)
		}
	}

	return nil
}

// Ensure FileTransfer implements FileTransferInterface
var _ FileTransferInterface = (*FileTransfer)(nil)

// TransferSessionInterface defines the interface for transfer sessions
type TransferSessionInterface interface {
	// UploadFile uploads a file within the session
	UploadFile(ctx context.Context, localPath, remotePath string, opts *TransferOptions) error

	// DownloadFile downloads a file within the session
	DownloadFile(ctx context.Context, remotePath, localPath string, opts *TransferOptions) error

	// GetStats returns current transfer statistics
	GetStats() *TransferStats

	// Close closes the transfer session
	Close() error
}

// Ensure TransferSession implements TransferSessionInterface
var _ TransferSessionInterface = (*TransferSession)(nil)
