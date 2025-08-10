package version

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// VersionCreateRequest represents the request for creating a new version
type VersionCreateRequest struct {
	AppID         string `json:"app_id"`
	VersionNumber string `json:"version_number"`
	Notes         string `json:"notes,omitempty"`
}

// VersionUpdateRequest represents the request for updating a version
type VersionUpdateRequest struct {
	VersionNumber string `json:"version_number,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

// VersionResponse represents a version response
type VersionResponse struct {
	ID            string    `json:"id"`
	AppID         string    `json:"app_id"`
	AppName       string    `json:"app_name,omitempty"`
	VersionNumber string    `json:"version_number"`
	Notes         string    `json:"notes"`
	HasZip        bool      `json:"has_zip"`
	ZipSize       int64     `json:"zip_size,omitempty"`
	Created       time.Time `json:"created"`
	Updated       time.Time `json:"updated"`
}

// VersionMetadata represents version metadata
type VersionMetadata struct {
	ID            string                 `json:"id"`
	VersionNumber string                 `json:"version_number"`
	Notes         string                 `json:"notes"`
	FileInfo      map[string]interface{} `json:"file_info,omitempty"`
	Validation    map[string]interface{} `json:"validation,omitempty"`
}

// listVersions handles the list versions endpoint
func listVersions(app core.App, e *core.RequestEvent) error {
	// Get optional filters
	appID := e.Request.URL.Query().Get("app_id")
	limit := 50 // Default limit

	if limitStr := e.Request.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var records []*core.Record
	var err error

	if appID != "" {
		// Filter by app ID
		records, err = app.FindRecordsByFilter("versions", "app_id = {:app_id}", "-created", limit, 0, map[string]any{
			"app_id": appID,
		})
	} else {
		// Get all versions
		records, err = app.FindRecordsByFilter("versions", "", "-created", limit, 0, nil)
	}

	if err != nil {
		app.Logger().Error("Failed to fetch versions", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch versions",
		})
	}

	// Convert records to response format
	versions := make([]VersionResponse, len(records))
	for i, record := range records {
		versions[i] = recordToVersionResponse(record, app)
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"versions": versions,
		"count":    len(versions),
	})
}

// createVersion handles the create version endpoint
func createVersion(app core.App, e *core.RequestEvent) error {
	var req VersionCreateRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.AppID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	if req.VersionNumber == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version number is required",
		})
	}

	// Validate version number format
	if err := validateVersionNumber(req.VersionNumber); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Verify app exists
	appRecord, err := app.FindRecordById("apps", req.AppID)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid app ID",
		})
	}

	// Check if version number is unique for this app
	existing, err := app.FindFirstRecordByFilter("versions", "app_id = {:app_id} && version_number = {:version}", map[string]any{
		"app_id":  req.AppID,
		"version": req.VersionNumber,
	})
	if err == nil && existing != nil {
		return e.JSON(http.StatusConflict, map[string]string{
			"error": "Version number already exists for this app",
		})
	}

	// Create new version record
	collection, err := app.FindCollectionByNameOrId("versions")
	if err != nil {
		app.Logger().Error("Failed to find versions collection", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	record := core.NewRecord(collection)
	record.Set("app_id", req.AppID)
	record.Set("version_number", req.VersionNumber)
	record.Set("notes", req.Notes)

	if err := app.Save(record); err != nil {
		app.Logger().Error("Failed to create version", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create version",
		})
	}

	app.Logger().Info("Version created successfully",
		"version_id", record.Id,
		"app_id", req.AppID,
		"version_number", req.VersionNumber)

	response := recordToVersionResponse(record, app)
	response.AppName = appRecord.GetString("name")

	return e.JSON(http.StatusCreated, response)
}

// getVersion handles the get single version endpoint
func getVersion(app core.App, e *core.RequestEvent) error {
	versionID := e.Request.PathValue("id")
	if versionID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version ID is required",
		})
	}

	// Get version record
	record, err := app.FindRecordById("versions", versionID)
	if err != nil {
		app.Logger().Error("Failed to find version", "id", versionID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Version not found",
		})
	}

	// Get app information
	appRecord, err := app.FindRecordById("apps", record.GetString("app_id"))
	if err != nil {
		app.Logger().Error("Failed to find app for version", "version_id", versionID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load app information",
		})
	}

	response := recordToVersionResponse(record, app)
	response.AppName = appRecord.GetString("name")

	return e.JSON(http.StatusOK, response)
}

// updateVersion handles the update version endpoint
func updateVersion(app core.App, e *core.RequestEvent) error {
	versionID := e.Request.PathValue("id")
	if versionID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version ID is required",
		})
	}

	var req VersionUpdateRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Get existing version record
	record, err := app.FindRecordById("versions", versionID)
	if err != nil {
		app.Logger().Error("Failed to find version", "id", versionID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Version not found",
		})
	}

	// Update fields if provided
	if req.VersionNumber != "" {
		if err := validateVersionNumber(req.VersionNumber); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}

		// Check if new version number is unique for this app
		existing, err := app.FindFirstRecordByFilter("versions", "app_id = {:app_id} && version_number = {:version} && id != {:id}", map[string]any{
			"app_id":  record.GetString("app_id"),
			"version": req.VersionNumber,
			"id":      versionID,
		})
		if err == nil && existing != nil {
			return e.JSON(http.StatusConflict, map[string]string{
				"error": "Version number already exists for this app",
			})
		}

		record.Set("version_number", req.VersionNumber)
	}

	if req.Notes != "" {
		record.Set("notes", req.Notes)
	}

	if err := app.Save(record); err != nil {
		app.Logger().Error("Failed to update version", "id", versionID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update version",
		})
	}

	app.Logger().Info("Version updated successfully", "version_id", versionID)

	return e.JSON(http.StatusOK, recordToVersionResponse(record, app))
}

// deleteVersion handles the delete version endpoint
func deleteVersion(app core.App, e *core.RequestEvent) error {
	versionID := e.Request.PathValue("id")
	if versionID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version ID is required",
		})
	}

	// Get version record to ensure it exists
	record, err := app.FindRecordById("versions", versionID)
	if err != nil {
		app.Logger().Error("Failed to find version", "id", versionID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Version not found",
		})
	}

	appID := record.GetString("app_id")
	versionNumber := record.GetString("version_number")

	// Check if this version is currently deployed
	appRecord, err := app.FindRecordById("apps", appID)
	if err == nil && appRecord.GetString("current_version") == versionNumber {
		return e.JSON(http.StatusConflict, map[string]string{
			"error": "Cannot delete currently deployed version",
		})
	}

	// Delete the version record
	if err := app.Delete(record); err != nil {
		app.Logger().Error("Failed to delete version", "id", versionID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete version",
		})
	}

	app.Logger().Info("Version deleted successfully",
		"version_id", versionID,
		"app_id", appID,
		"version_number", versionNumber)

	return e.JSON(http.StatusOK, map[string]interface{}{
		"message":    "Version deleted successfully",
		"version_id": versionID,
	})
}

// uploadVersionZip handles uploading binary and public folder files for a version
func uploadVersionZip(app core.App, e *core.RequestEvent) error {
	versionID := e.Request.PathValue("id")
	if versionID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version ID is required",
		})
	}

	// Get version record
	record, err := app.FindRecordById("versions", versionID)
	if err != nil {
		app.Logger().Error("Failed to find version", "id", versionID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Version not found",
		})
	}

	// Parse multipart form
	if err := e.Request.ParseMultipartForm(157286400); err != nil { // 150MB max
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to parse multipart form",
		})
	}

	// Get uploaded binary file
	binaryFile, binaryHeader, err := e.Request.FormFile("pocketbase_binary")
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "PocketBase binary file is required",
		})
	}
	defer binaryFile.Close()

	// Get uploaded public folder files
	publicFiles := e.Request.MultipartForm.File["pb_public_files"]
	if len(publicFiles) == 0 {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "pb_public folder files are required",
		})
	}

	// Validate file sizes
	if binaryHeader.Size > 104857600 { // 100MB max for binary
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Binary file size exceeds 100MB limit",
		})
	}

	// Calculate total size of public files and validate
	var totalPublicSize int64
	for _, fileHeader := range publicFiles {
		totalPublicSize += fileHeader.Size
	}

	if totalPublicSize > 52428800 { // 50MB max for public folder
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Public folder total size exceeds 50MB limit",
		})
	}

	// Create deployment ZIP in memory
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	// Add binary file to ZIP
	binaryWriter, err := zipWriter.Create("pocketbase")
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create deployment package",
		})
	}

	if _, err := io.Copy(binaryWriter, binaryFile); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to add binary to deployment package",
		})
	}

	// Add public folder files to ZIP
	for _, fileHeader := range publicFiles {
		// Open the uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to read uploaded public file",
			})
		}

		// Create file in deployment ZIP under pb_public/
		// The filename may contain path separators for folder structure
		deploymentPath := fmt.Sprintf("pb_public/%s", fileHeader.Filename)
		writer, err := zipWriter.Create(deploymentPath)
		if err != nil {
			file.Close()
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create deployment package",
			})
		}

		// Copy file content
		if _, err := io.Copy(writer, file); err != nil {
			file.Close()
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create deployment package",
			})
		}
		file.Close()
	}

	// Close ZIP writer
	if err := zipWriter.Close(); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to finalize deployment package",
		})
	}

	// Create a multipart file header for the deployment ZIP
	deploymentFilename := fmt.Sprintf("deployment_%s_%d.zip", record.GetString("version_number"), time.Now().Unix())

	record.Set("deployment_zip", deploymentFilename)

	if err := app.Save(record); err != nil {
		app.Logger().Error("Failed to save version record", "version_id", versionID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save deployment package",
		})
	}

	// Save the actual file content using PocketBase filesystem
	filesystem, err := app.NewFilesystem()
	if err != nil {
		app.Logger().Error("Failed to get filesystem", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save deployment package",
		})
	}
	defer filesystem.Close()

	// Save ZIP content to filesystem
	fileKey := record.BaseFilesPath() + "/" + deploymentFilename
	if err := filesystem.Upload(zipBuffer.Bytes(), fileKey); err != nil {
		app.Logger().Error("Failed to save deployment zip to filesystem", "version_id", versionID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save deployment package",
		})
	}

	app.Logger().Info("Version files uploaded successfully",
		"version_id", versionID,
		"binary_file", binaryHeader.Filename,
		"binary_size", binaryHeader.Size,
		"public_files_count", len(publicFiles),
		"public_total_size", totalPublicSize,
		"deployment_size", zipBuffer.Len())

	return e.JSON(http.StatusOK, map[string]interface{}{
		"message":            "Version files uploaded successfully",
		"version_id":         versionID,
		"binary_file":        binaryHeader.Filename,
		"binary_size":        binaryHeader.Size,
		"public_files_count": len(publicFiles),
		"public_total_size":  totalPublicSize,
		"deployment_file":    deploymentFilename,
		"deployment_size":    zipBuffer.Len(),
		"uploaded_at":        time.Now().UTC(),
	})
}

// downloadVersionZip handles downloading a deployment zip for a version
func downloadVersionZip(app core.App, e *core.RequestEvent) error {
	versionID := e.Request.PathValue("id")
	if versionID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version ID is required",
		})
	}

	// Get version record
	record, err := app.FindRecordById("versions", versionID)
	if err != nil {
		app.Logger().Error("Failed to find version", "id", versionID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Version not found",
		})
	}

	// Check if deployment zip exists
	deploymentZip := record.GetString("deployment_zip")
	if deploymentZip == "" {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "No deployment zip found for this version",
		})
	}

	// Get file from filesystem
	filesystem, err := app.NewFilesystem()
	if err != nil {
		app.Logger().Error("Failed to get filesystem", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to access file system",
		})
	}
	defer filesystem.Close()

	// Serve the file
	serveKey := record.BaseFilesPath() + "/" + deploymentZip
	return filesystem.Serve(e.Response, e.Request, serveKey, deploymentZip)
}

// listAppVersions handles listing versions for a specific app
func listAppVersions(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("app_id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	// Verify app exists
	appRecord, err := app.FindRecordById("apps", appID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	// Get versions for this app
	records, err := app.FindRecordsByFilter("versions", "app_id = {:app_id}", "-created", 0, 0, map[string]any{
		"app_id": appID,
	})
	if err != nil {
		app.Logger().Error("Failed to fetch app versions", "app_id", appID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch versions",
		})
	}

	// Convert records to response format
	versions := make([]VersionResponse, len(records))
	for i, record := range records {
		versions[i] = recordToVersionResponse(record, app)
		versions[i].AppName = appRecord.GetString("name")
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"app_id":   appID,
		"app_name": appRecord.GetString("name"),
		"versions": versions,
		"count":    len(versions),
	})
}

// createAppVersion handles creating a version for a specific app
func createAppVersion(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("app_id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	var req VersionCreateRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Override app_id with path parameter
	req.AppID = appID

	// Delegate to main create function
	return createVersion(app, e)
}

// validateVersion handles validating a deployment zip
func validateVersion(app core.App, e *core.RequestEvent) error {
	versionID := e.Request.PathValue("id")
	if versionID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version ID is required",
		})
	}

	// Get version record
	record, err := app.FindRecordById("versions", versionID)
	if err != nil {
		app.Logger().Error("Failed to find version", "id", versionID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Version not found",
		})
	}

	// Check if deployment zip exists
	deploymentZip := record.GetString("deployment_zip")
	if deploymentZip == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "No deployment zip found for this version",
		})
	}

	// TODO: Implement actual zip validation
	// This would include:
	// 1. Checking if zip is valid
	// 2. Verifying required files (pocketbase binary, pb_public folder)
	// 3. Checking file permissions
	// 4. Validating structure

	validation := map[string]interface{}{
		"valid":      true,
		"checked_at": time.Now().UTC(),
		"checks":     []string{"zip_structure", "required_files", "permissions"},
		"warnings":   []string{},
		"errors":     []string{},
		"message":    "Validation not yet implemented - assuming valid",
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"version_id": versionID,
		"validation": validation,
	})
}

// getVersionMetadata handles getting version metadata
func getVersionMetadata(app core.App, e *core.RequestEvent) error {
	versionID := e.Request.PathValue("id")
	if versionID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version ID is required",
		})
	}

	// Get version record
	record, err := app.FindRecordById("versions", versionID)
	if err != nil {
		app.Logger().Error("Failed to find version", "id", versionID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Version not found",
		})
	}

	metadata := VersionMetadata{
		ID:            record.Id,
		VersionNumber: record.GetString("version_number"),
		Notes:         record.GetString("notes"),
	}

	// Add file info if deployment zip exists
	deploymentZip := record.GetString("deployment_zip")
	if deploymentZip != "" {
		metadata.FileInfo = map[string]interface{}{
			"filename": deploymentZip,
			"has_file": true,
		}
	} else {
		metadata.FileInfo = map[string]interface{}{
			"has_file": false,
		}
	}

	return e.JSON(http.StatusOK, metadata)
}

// updateVersionMetadata handles updating version metadata
func updateVersionMetadata(app core.App, e *core.RequestEvent) error {
	versionID := e.Request.PathValue("id")
	if versionID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version ID is required",
		})
	}

	var req map[string]interface{}
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Get version record
	record, err := app.FindRecordById("versions", versionID)
	if err != nil {
		app.Logger().Error("Failed to find version", "id", versionID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Version not found",
		})
	}

	// Update allowed metadata fields
	if notes, ok := req["notes"].(string); ok {
		record.Set("notes", notes)
	}

	if err := app.Save(record); err != nil {
		app.Logger().Error("Failed to update version metadata", "id", versionID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update version metadata",
		})
	}

	app.Logger().Info("Version metadata updated successfully", "version_id", versionID)

	return e.JSON(http.StatusOK, map[string]interface{}{
		"message":    "Metadata updated successfully",
		"version_id": versionID,
	})
}

// Helper functions

// recordToVersionResponse converts a version record to response format
func recordToVersionResponse(record *core.Record, app core.App) VersionResponse {
	response := VersionResponse{
		ID:            record.Id,
		AppID:         record.GetString("app_id"),
		VersionNumber: record.GetString("version_number"),
		Notes:         record.GetString("notes"),
		Created:       record.GetDateTime("created").Time(),
		Updated:       record.GetDateTime("updated").Time(),
	}

	// Check if deployment zip exists
	deploymentZip := record.GetString("deployment_zip")
	response.HasZip = deploymentZip != ""

	return response
}

// validateVersionNumber validates version number format
func validateVersionNumber(version string) error {
	if len(version) < 1 || len(version) > 50 {
		return fmt.Errorf("version number must be between 1 and 50 characters")
	}

	// Allow semantic versioning and other common patterns
	// This is a basic validation - could be enhanced
	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("version number cannot be empty or just whitespace")
	}

	return nil
}
