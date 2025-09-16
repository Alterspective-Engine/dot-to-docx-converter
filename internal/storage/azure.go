package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	log "github.com/sirupsen/logrus"
)

// AzureStorage implements Storage using Azure Blob Storage
type AzureStorage struct {
	client       *azblob.Client
	containerName string
	localCache   string
}

// NewAzureStorage creates a new Azure Blob Storage instance
func NewAzureStorage(connectionString string) (*AzureStorage, error) {
	if connectionString == "" {
		return nil, fmt.Errorf("Azure Storage connection string is required")
	}

	// Create client
	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure Storage client: %w", err)
	}

	// Create local cache directory
	cacheDir := filepath.Join(os.TempDir(), "azure-cache")
	os.MkdirAll(cacheDir, 0755)

	log.Info("Connected to Azure Blob Storage")

	return &AzureStorage{
		client:        client,
		containerName: "conversions",
		localCache:    cacheDir,
	}, nil
}

// Upload uploads a file to Azure Blob Storage
func (s *AzureStorage) Upload(ctx context.Context, localPath, remotePath string) error {
	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Normalize blob name
	blobName := strings.ReplaceAll(remotePath, "\\", "/")

	// Upload to blob
	_, err = s.client.UploadFile(ctx, s.containerName, blobName, file, nil)
	if err != nil {
		return fmt.Errorf("failed to upload to Azure: %w", err)
	}

	log.Debugf("Uploaded %s to Azure Blob Storage as %s", localPath, blobName)
	return nil
}

// Download downloads a file from Azure Blob Storage
func (s *AzureStorage) Download(ctx context.Context, remotePath string) (string, error) {
	// Normalize blob name
	blobName := strings.ReplaceAll(remotePath, "\\", "/")

	// Create local cache path
	localPath := filepath.Join(s.localCache, filepath.Base(remotePath))

	// Download from blob
	file, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	// Get blob client
	blobClient := s.client.ServiceClient().NewContainerClient(s.containerName).NewBlobClient(blobName)

	// Download blob
	get, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		os.Remove(localPath)
		return "", fmt.Errorf("failed to download from Azure: %w", err)
	}

	// Copy to local file
	reader := get.Body
	defer reader.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		os.Remove(localPath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	log.Debugf("Downloaded %s from Azure Blob Storage to %s", blobName, localPath)
	return localPath, nil
}

// List lists files in Azure Blob Storage
func (s *AzureStorage) List(ctx context.Context, prefix string) ([]string, error) {
	// Normalize prefix
	prefix = strings.ReplaceAll(prefix, "\\", "/")

	var files []string
	containerClient := s.client.ServiceClient().NewContainerClient(s.containerName)

	// List blobs with prefix
	pager := containerClient.NewListBlobsFlatPager(&azblob.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list blobs: %w", err)
		}

		for _, blob := range resp.Segment.BlobItems {
			if blob.Name != nil {
				files = append(files, *blob.Name)
			}
		}
	}

	return files, nil
}

// ReadFile reads file contents from Azure Blob Storage
func (s *AzureStorage) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// Download to temp location first
	localPath, err := s.Download(ctx, path)
	if err != nil {
		return nil, err
	}
	defer os.Remove(localPath)

	return os.ReadFile(localPath)
}

// WriteFile writes data to a file (via local cache then upload)
func (s *AzureStorage) WriteFile(path string, data []byte) error {
	// Write to local cache first
	localPath := filepath.Join(s.localCache, filepath.Base(path))

	if err := os.WriteFile(localPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write local file: %w", err)
	}

	// Upload to Azure
	ctx := context.Background()
	if err := s.Upload(ctx, localPath, path); err != nil {
		os.Remove(localPath)
		return err
	}

	os.Remove(localPath)
	return nil
}

// Delete deletes a file from Azure Blob Storage
func (s *AzureStorage) Delete(ctx context.Context, path string) error {
	// Normalize blob name
	blobName := strings.ReplaceAll(path, "\\", "/")

	// Delete blob
	blobClient := s.client.ServiceClient().NewContainerClient(s.containerName).NewBlobClient(blobName)
	_, err := blobClient.Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete from Azure: %w", err)
	}

	log.Debugf("Deleted %s from Azure Blob Storage", blobName)
	return nil
}

// GetLocalPath returns a local cache path for the given storage path
func (s *AzureStorage) GetLocalPath(path string) string {
	return filepath.Join(s.localCache, filepath.Base(path))
}

// EnsureDirectory ensures a directory exists (no-op for blob storage)
func (s *AzureStorage) EnsureDirectory(path string) error {
	// Azure Blob Storage doesn't have real directories
	// Just ensure local cache directory exists
	localDir := filepath.Join(s.localCache, path)
	return os.MkdirAll(localDir, 0755)
}

// Cleanup removes temporary local files
func (s *AzureStorage) Cleanup(localPath string) error {
	// Only clean up cache files
	if strings.HasPrefix(localPath, s.localCache) {
		return os.Remove(localPath)
	}
	return nil
}