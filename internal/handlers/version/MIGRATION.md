# Version Handlers Migration Guide

## Overview

Migrate version handlers from basic file operations to the modern tunnel/tracer/models architecture with enhanced file management, validation, and deployment package handling.

## Current State Analysis

### Current Issues
- Direct database record manipulation without models abstraction
- Basic file upload/download without validation pipeline
- No structured tracing for file operations
- Limited deployment package validation
- Basic error handling without categorization
- Manual ZIP creation and extraction
- No file integrity checks or checksums
- Missing version lifecycle management
- No integration with deployment pipeline
- Basic metadata handling

### Files to Migrate
- `handlers.go` - Handler registration (needs dependency injection)
- `management.go` - Version CRUD, file operations, and validation

## Migration Strategy

### Phase 1: Models Integration
Replace direct database operations with models package abstractions.

### Phase 2: File Management Enhancement
Integrate with tunnel.FileManager for advanced file operations.

### Phase 3: Deployment Pipeline Integration
Connect version operations with deployment managers.

## File-by-File Migration

### `handlers.go` - Handler Registration

**Current:**
```go
func RegisterVersionHandlers(app core.App, group *router.RouterGroup[*core.RequestEvent]) {
    // Direct handler functions
}
```

**Target:**
```go
type VersionHandlers struct {
    executor   tunnel.Executor
    deployMgr  tunnel.DeploymentManager
    tracer     tracer.ServiceTracer
    validator  tunnel.PackageValidator
}

func NewVersionHandlers(
    executor tunnel.Executor,
    fileMgr tunnel.FileManager,
    deployMgr tunnel.DeploymentManager,
    tracerFactory tracer.TracerFactory,
) *VersionHandlers {
    return &VersionHandlers{
        executor:  executor,
        deployMgr: deployMgr,
        tracer:    tracerFactory.CreateServiceTracer(),
        validator: tunnel.NewPackageValidator(),
    }
}

func (h *VersionHandlers) RegisterRoutes(group *router.RouterGroup[*core.RequestEvent]) {
    // Handler methods with dependency injection
}
```

### `management.go` - Version Operations

#### Current Issues
- Direct record manipulation without validation
- Basic file upload without integrity checks
- Limited ZIP validation
- Manual multipart form handling
- No file operation tracing
- Basic metadata management

#### Migration Changes

**Version Creation:**
```go
// BEFORE
func createVersion(app core.App, e *core.RequestEvent) error {
    // Direct record creation
    record := core.NewRecord(collection)
    record.Set("app_id", req.AppID)
    record.Set("version_number", req.VersionNumber)
    app.Save(record)
}

// AFTER
func (h *VersionHandlers) createVersion(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "version", "create")
    defer span.End()
    
    var req VersionCreateRequest
    if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
        return handleValidationError(e, err, "Invalid request body")
    }
    
    // Validate request using models
    if err := h.validateVersionRequest(req); err != nil {
        span.EndWithError(err)
        return handleValidationError(e, err, "Version validation failed")
    }
    
    // Verify app exists and get model
    appModel, err := models.GetApp(app, req.AppID)
    if err != nil {
        span.EndWithError(err)
        return handleAppError(e, err, "App not found")
    }
    
    // Check version uniqueness
    if exists, err := models.VersionExists(app, req.AppID, req.VersionNumber); err != nil {
        span.EndWithError(err)
        return handleDatabaseError(e, err, "Failed to check version uniqueness")
    } else if exists {
        return e.JSON(http.StatusConflict, ConflictErrorResponse{
            Error:      "Version already exists",
            Details:    fmt.Sprintf("Version %s already exists for app %s", req.VersionNumber, appModel.Name),
            Suggestion: "Use a different version number or update existing version",
            Code:       "VERSION_EXISTS",
        })
    }
    
    // Create version through models
    versionModel := models.NewVersion()
    versionModel.AppID = req.AppID
    versionModel.VersionNumber = req.VersionNumber
    versionModel.Notes = req.Notes
    
    if err := models.SaveVersion(app, versionModel); err != nil {
        span.EndWithError(err)
        return handleDatabaseError(e, err, "Failed to create version")
    }
    
    span.SetFields(tracer.Fields{
        "version.id":     versionModel.ID,
        "version.number": versionModel.VersionNumber,
        "app.id":        appModel.ID,
        "app.name":      appModel.Name,
    })
    
    span.Event("version_created")
    
    return e.JSON(http.StatusCreated, VersionResponse{
        ID:            versionModel.ID,
        AppID:         versionModel.AppID,
        AppName:       appModel.Name,
        VersionNumber: versionModel.VersionNumber,
        Notes:         versionModel.Notes,
        HasZip:        false,
        Created:       versionModel.Created,
        Updated:       versionModel.Updated,
    })
}
```

**Enhanced File Upload:**
```go
// BEFORE
func uploadVersionZip(app core.App, e *core.RequestEvent) error {
    // Manual multipart form parsing
    binaryFile, binaryHeader, err := e.Request.FormFile("pocketbase_binary")
    publicFiles := e.Request.MultipartForm.File["pb_public_files"]
    
    // Manual ZIP creation
    var zipBuffer bytes.Buffer
    zipWriter := zip.NewWriter(&zipBuffer)
    // Manual file processing
}

// AFTER
func (h *VersionHandlers) uploadVersionPackage(app core.App, e *core.RequestEvent) error {
    span := h.fileTracer.TraceFileTransfer(e.Request.Context(), "upload", "deployment_package", 0)
    defer span.End()
    
    versionModel, err := models.GetVersion(app, versionID)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "Version not found")
    }
    
    // Parse upload request using file manager
    uploadRequest, err := h.fileMgr.ParseUploadRequest(e.Request, tunnel.UploadConfig{
        MaxFileSize:      104857600, // 100MB binary limit
        MaxTotalSize:     157286400, // 150MB total limit
        AllowedTypes:     []string{"application/octet-stream", "application/zip"},
        RequiredFiles:    []string{"pocketbase_binary"},
        OptionalFiles:    []string{"pb_public_files"},
        ValidateStructure: true,
    })
    
    if err != nil {
        span.EndWithError(err)
        return handleUploadError(e, err, "Invalid upload request")
    }
    
    // Validate binary file
    binaryValidation, err := h.validator.ValidatePocketBaseBinary(uploadRequest.Files["pocketbase_binary"])
    if err != nil {
        span.EndWithError(err)
        return handleValidationError(e, err, "Binary validation failed")
    }
    
    if !binaryValidation.Valid {
        return e.JSON(http.StatusBadRequest, ValidationErrorResponse{
            Error:   "Invalid PocketBase binary",
            Issues:  binaryValidation.Issues,
            Suggestions: binaryValidation.Suggestions,
            Code:    "INVALID_BINARY",
        })
    }
    
    // Validate public files structure
    publicValidation, err := h.validator.ValidatePublicFiles(uploadRequest.Files["pb_public_files"])
    if err != nil {
        span.EndWithError(err)
        return handleValidationError(e, err, "Public files validation failed")
    }
    
    // Create deployment package using file manager
    packageConfig := tunnel.PackageConfig{
        VersionID:     versionModel.ID,
        AppName:       versionModel.GetAppName(),
        Version:       versionModel.VersionNumber,
        BinaryFile:    uploadRequest.Files["pocketbase_binary"],
        PublicFiles:   uploadRequest.Files["pb_public_files"],
        Compression:   tunnel.CompressionBest,
        IncludeChecksum: true,
        ValidateIntegrity: true,
    }
    
    // Create package with progress tracking
    progressChan := make(chan tunnel.PackageProgress, 10)
    // Create deployment package using executor
    packagePath := filepath.Join("/tmp", fmt.Sprintf("version_%d.zip", versionModel.ID))
    
    // Use executor to create package via shell commands
    createCmd := tunnel.Command{
        Cmd:     fmt.Sprintf("cd %s && zip -r %s *", uploadDir, packagePath),
        Timeout: 5 * time.Minute,
    }
    
    result, err := h.executor.RunCommand(e.Request.Context(), createCmd)
    
    if err != nil {
        span.EndWithError(err)
        return handlePackageError(e, err, "Failed to create deployment package")
    }
    
    // Update version model with package information
    versionModel.DeploymentZip = packageResult.Filename
    versionModel.PackageSize = packageResult.Size
    versionModel.Checksum = packageResult.Checksum
    versionModel.ValidationStatus = "validated"
    
    if err := models.SaveVersion(app, versionModel); err != nil {
        span.EndWithError(err)
        return handleDatabaseError(e, err, "Failed to update version with package info")
    }
    
    span.SetFields(tracer.Fields{
        "version.id":          versionModel.ID,
        "package.size":        packageResult.Size,
        "package.files":       len(packageResult.Files),
        "package.compression": packageResult.CompressionRatio,
        "package.checksum":    packageResult.Checksum,
    })
    
    span.Event("package_created")
    
    return e.JSON(http.StatusOK, PackageUploadResponse{
        VersionID:       versionModel.ID,
        PackageFile:     packageResult.Filename,
        PackageSize:     packageResult.Size,
        Checksum:       packageResult.Checksum,
        BinarySize:     packageResult.BinarySize,
        PublicFilesCount: len(packageResult.PublicFiles),
        CompressionRatio: packageResult.CompressionRatio,
        ValidationStatus: "validated",
        UploadedAt:      time.Now().UTC(),
    })
}
```

**Enhanced File Download:**
```go
// BEFORE
func downloadVersionZip(app core.App, e *core.RequestEvent) error {
    // Basic file serving
    filesystem, err := app.NewFilesystem()
    serveKey := record.BaseFilesPath() + "/" + deploymentZip
    return filesystem.Serve(e.Response, e.Request, serveKey, deploymentZip)
}

// AFTER
func (h *VersionHandlers) downloadVersionPackage(app core.App, e *core.RequestEvent) error {
    span := h.fileTracer.TraceFileTransfer(e.Request.Context(), "download", "deployment_package", 0)
    defer span.End()
    
    versionModel, err := models.GetVersion(app, versionID)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "Version not found")
    }
    
    if !versionModel.HasDeploymentZip() {
        return e.JSON(http.StatusNotFound, NotFoundErrorResponse{
            Error:      "No deployment package found",
            Details:    "Version has no uploaded deployment package",
            Suggestion: "Upload deployment package first",
            Code:       "NO_PACKAGE",
        })
    }
    
    // Validate package exists and get info
    packagePath := versionModel.GetDeploymentZipPath()
    
    statCmd := tunnel.Command{
        Cmd:     fmt.Sprintf("stat %s", packagePath),
        Timeout: 10 * time.Second,
    }
    
    _, err = h.executor.RunCommand(e.Request.Context(), statCmd)
    if err != nil {
        span.EndWithError(err)
        return handleIntegrityError(e, err, "Package file not found")
    }
    
    // Prepare download with metadata
    downloadConfig := tunnel.DownloadConfig{
        PackagePath:    packagePath,
        Filename:      versionModel.GetDownloadFilename(),
        ContentType:   "application/zip",
        CacheControl:  "private, max-age=3600",
        LastModified:  versionModel.Updated,
        ETag:          versionModel.Checksum,
        Disposition:   "attachment",
    }
    
    // Record download metrics
    span.SetFields(tracer.Fields{
        "version.id":       versionModel.ID,
        "package.size":     versionModel.PackageSize,
        "package.filename": downloadConfig.Filename,
        "download.client":  getClientIP(e.Request),
    })
    
    // Transfer file from server using executor
    transferConfig := tunnel.FileTransfer{
        LocalPath:  packagePath,
        RemotePath: versionModel.GetDeploymentZipPath(),
        Direction:  tunnel.TransferDownload,
        Progress:   true,
    }
    
    err = h.executor.TransferFile(e.Request.Context(), transferConfig)
    if err != nil {
        span.EndWithError(err)
        return handleDownloadError(e, err, "File transfer failed")
    }
    
    span.Event("package_downloaded")
    
    // Update download statistics
    versionModel.IncrementDownloadCount()
    models.SaveVersion(app, versionModel)
    
    return nil
}
```

**Package Validation:**
```go
// BEFORE
func validateVersion(app core.App, e *core.RequestEvent) error {
    // TODO: Implement actual zip validation
    validation := map[string]any{
        "valid":   true,
        "message": "Validation not yet implemented - assuming valid",
    }
}

// AFTER
func (h *VersionHandlers) validateVersionPackage(app core.App, e *core.RequestEvent) error {
    span := h.fileTracer.TraceFileTransfer(e.Request.Context(), "validate", "deployment_package", 0)
    defer span.End()
    
    versionModel, err := models.GetVersion(app, versionID)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "Version not found")
    }
    
    if !versionModel.HasDeploymentZip() {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "No deployment package to validate",
        })
    }
    
    // Comprehensive package validation
    validationConfig := tunnel.ValidationConfig{
        CheckStructure:     true,
        CheckBinary:        true,
        CheckPermissions:   true,
        CheckDependencies:  true,
        CheckSecurity:     true,
        GenerateReport:    true,
        DeepScan:          e.Request.URL.Query().Get("deep") == "true",
    }
    
    validationResult, err := h.validator.ValidateDeploymentPackage(
        e.Request.Context(),
        versionModel.GetDeploymentZipPath(),
        validationConfig,
    )
    
    if err != nil {
        span.EndWithError(err)
        return handleValidationError(e, err, "Package validation failed")
    }
    
    // Update version with validation results
    versionModel.ValidationStatus = validationResult.Status
    versionModel.ValidationReport = validationResult.Report
    versionModel.LastValidated = time.Now()
    
    if err := models.SaveVersion(app, versionModel); err != nil {
        h.tracer.RecordError(span, err, "failed to save validation results")
    }
    
    response := PackageValidationResponse{
        VersionID:        versionModel.ID,
        ValidationStatus: validationResult.Status,
        Valid:           validationResult.Valid,
        Warnings:        validationResult.Warnings,
        Errors:          validationResult.Errors,
        SecurityIssues:   validationResult.SecurityIssues,
        Recommendations: validationResult.Recommendations,
        Report:          validationResult.Report,
        ValidatedAt:     time.Now().UTC(),
    }
    
    span.SetFields(tracer.Fields{
        "version.id":           versionModel.ID,
        "validation.valid":     validationResult.Valid,
        "validation.warnings":  len(validationResult.Warnings),
        "validation.errors":    len(validationResult.Errors),
        "validation.security":  len(validationResult.SecurityIssues),
    })
    
    return e.JSON(http.StatusOK, response)
}
```

**Enhanced Metadata Management:**
```go
// BEFORE
func getVersionMetadata(app core.App, e *core.RequestEvent) error {
    metadata := VersionMetadata{
        ID:            record.Id,
        VersionNumber: record.GetString("version_number"),
        Notes:         record.GetString("notes"),
    }
    
    if deploymentZip != "" {
        metadata.FileInfo = map[string]any{"filename": deploymentZip}
    }
}

// AFTER
func (h *VersionHandlers) getVersionMetadata(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "version", "metadata")
    defer span.End()
    
    versionModel, err := models.GetVersion(app, versionID)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "Version not found")
    }
    
    // Get comprehensive metadata
    metadata := h.buildVersionMetadata(e.Request.Context(), versionModel)
    
    // Add deployment compatibility info
    if versionModel.HasDeploymentZip() {
        compatibility, err := h.deployMgr.CheckVersionCompatibility(e.Request.Context(), versionModel.ID)
        if err == nil {
            metadata.Compatibility = compatibility
        }
    }
    
    // Add usage statistics
    usage := h.getVersionUsageStats(versionModel.ID)
    metadata.Usage = usage
    
    span.SetFields(tracer.Fields{
        "version.id":          versionModel.ID,
        "version.has_package": versionModel.HasDeploymentZip(),
        "version.deployments": usage.DeploymentCount,
    })
    
    return e.JSON(http.StatusOK, metadata)
}

func (h *VersionHandlers) buildVersionMetadata(ctx context.Context, version *models.Version) *EnhancedVersionMetadata {
    span := h.fileTracer.TraceFileTransfer(ctx, "metadata", "analysis", 0)
    defer span.End()
    
    metadata := &EnhancedVersionMetadata{
        ID:            version.ID,
        VersionNumber: version.VersionNumber,
        Notes:         version.Notes,
        Created:       version.Created,
        Updated:       version.Updated,
    }
    
    if version.HasDeploymentZip() {
        // Get file information using executor
        statCmd := tunnel.Command{
            Cmd:     fmt.Sprintf("stat -c '%%s,%%Y' %s", version.GetDeploymentZipPath()),
            Timeout: 10 * time.Second,
        }
        
        result, err := h.executor.RunCommand(context.Background(), statCmd)
        if err == nil {
            parts := strings.Split(strings.TrimSpace(result.Output), ",")
            if len(parts) == 2 {
                size, _ := strconv.ParseInt(parts[0], 10, 64)
                modTime, _ := strconv.ParseInt(parts[1], 10, 64)
                
                metadata.FileInfo = FileInfo{
                    Filename:     filepath.Base(version.GetDeploymentZipPath()),
                    Size:         size,
                    LastModified: time.Unix(modTime, 0),
                }
            }
        }
        
        // Get package contents using executor
        listCmd := tunnel.Command{
            Cmd:     fmt.Sprintf("unzip -l %s", version.GetDeploymentZipPath()),
            Timeout: 30 * time.Second,
        }
        
        result, err := h.executor.RunCommand(context.Background(), listCmd)
        if err == nil {
            metadata.PackageContents = parseZipListing(result.Output)
        }
        
        // Get validation history
        validationHistory := h.getValidationHistory(version.ID)
        metadata.ValidationHistory = validationHistory
    }
    
    span.SetField("metadata.complete", true)
    return metadata
}
```

## Enhanced Features to Implement

### Version Lifecycle Management
```go
func (h *VersionHandlers) getVersionLifecycle(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "version", "lifecycle")
    defer span.End()
    
    versionModel, err := models.GetVersion(app, versionID)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "Version not found")
    }
    
    // Get comprehensive lifecycle information
    lifecycle := VersionLifecycle{
        VersionID:     versionModel.ID,
        CurrentPhase:  h.determineVersionPhase(versionModel),
        Deployments:   h.getVersionDeployments(versionModel.ID),
        Health:        h.getVersionHealth(e.Request.Context(), versionModel),
        Metrics:       h.getVersionMetrics(versionModel.ID),
        Timeline:      h.getVersionTimeline(versionModel.ID),
        Dependencies:  h.getVersionDependencies(e.Request.Context(), versionModel),
        Recommendations: h.getVersionRecommendations(versionModel),
    }
    
    span.SetFields(tracer.Fields{
        "version.id":          versionModel.ID,
        "lifecycle.phase":     lifecycle.CurrentPhase,
        "lifecycle.deployments": len(lifecycle.Deployments),
        "lifecycle.health":    lifecycle.Health.Overall,
    })
    
    return e.JSON(http.StatusOK, lifecycle)
}
```

### Version Comparison and Diff
```go
func (h *VersionHandlers) compareVersions(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "version", "compare")
    defer span.End()
    
    fromVersionID := e.Request.URL.Query().Get("from")
    toVersionID := e.Request.URL.Query().Get("to")
    
    if fromVersionID == "" || toVersionID == "" {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Both 'from' and 'to' version IDs are required",
        })
    }
    
    fromVersion, err := models.GetVersion(app, fromVersionID)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "From version not found")
    }
    
    toVersion, err := models.GetVersion(app, toVersionID)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "To version not found")
    }
    
    // Ensure versions belong to same app
    if fromVersion.AppID != toVersion.AppID {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Versions must belong to the same app",
        })
    }
    
    // Perform comprehensive comparison
    comparisonConfig := tunnel.ComparisonConfig{
        IncludeFiles:    e.Request.URL.Query().Get("include_files") == "true",
        IncludeMetrics:  e.Request.URL.Query().Get("include_metrics") == "true",
        IncludeSecurity: e.Request.URL.Query().Get("include_security") == "true",
        DeepAnalysis:   e.Request.URL.Query().Get("deep") == "true",
    }
    
    // Compare versions using executor commands
    comparison, err := h.compareVersionPackages(e.Request.Context(), fromVersion, toVersion, comparisonConfig)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "Version comparison failed")
    }
    
    span.SetFields(tracer.Fields{
        "comparison.from":     fromVersionID,
        "comparison.to":       toVersionID,
        "comparison.changes":  len(comparison.Changes),
        "comparison.type":     comparison.Type,
    })
    
    return e.JSON(http.StatusOK, comparison)
}
```

### Package Optimization
```go
func (h *VersionHandlers) optimizeVersionPackage(app core.App, e *core.RequestEvent) error {
    span := h.fileTracer.TraceFileTransfer(e.Request.Context(), "optimize", "deployment_package", 0)
    defer span.End()
    
    versionModel, err := models.GetVersion(app, versionID)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "Version not found")
    }
    
    if !versionModel.HasDeploymentZip() {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "No deployment package to optimize",
        })
    }
    
    // Get optimization configuration
    optimizationConfig := tunnel.OptimizationConfig{
        Compression:      tunnel.CompressionBest,
        RemoveUnused:     e.Request.URL.Query().Get("remove_unused") == "true",
        OptimizeImages:   e.Request.URL.Query().Get("optimize_images") == "true",
        MinifyAssets:     e.Request.URL.Query().Get("minify_assets") == "true",
        CreateBackup:     true,
    }
    
    // Perform optimization using executor
    optimizeCmd := tunnel.Command{
        Cmd:     fmt.Sprintf("gzip -9 %s && mv %s.gz %s", versionModel.GetDeploymentZipPath(), versionModel.GetDeploymentZipPath(), versionModel.GetDeploymentZipPath()),
        Timeout: 10 * time.Minute,
    }
    
    result, err := h.executor.RunCommand(e.Request.Context(), optimizeCmd)
    if err != nil {
        span.EndWithError(err)
        return handleOptimizationError(e, err, "Package optimization failed")
    }
    
    // Update version with optimization results
    versionModel.PackageSize = optimizationResult.NewSize
    versionModel.Checksum = optimizationResult.NewChecksum
    versionModel.OptimizationApplied = true
    versionModel.OptimizationSavings = optimizationResult.SpaceSaved
    
    if err := models.SaveVersion(app, versionModel); err != nil {
        h.tracer.RecordError(span, err, "failed to save optimization results")
    }
    
    span.SetFields(tracer.Fields{
        "version.id":            versionModel.ID,
        "optimization.original": optimizationResult.OriginalSize,
        "optimization.new":      optimizationResult.NewSize,
        "optimization.saved":    optimizationResult.SpaceSaved,
        "optimization.ratio":    optimizationResult.CompressionRatio,
    })
    
    return e.JSON(http.StatusOK, optimizationResult)
}
```

### Version Analytics and Insights
```go
func (h *VersionHandlers) getVersionAnalytics(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "version", "analytics")
    defer span.End()
    
    // Parse analytics parameters
    params := parseAnalyticsParams(e.Request.URL.Query())
    
    // Get version analytics from models
    analytics, err := models.GetVersionAnalytics(app, params)
    if err != nil {
        span.EndWithError(err)
        return handleAnalyticsError(e, err, "Failed to get version analytics")
    }
    
    // Enhance with deployment correlation
    deploymentCorrelation := h.getDeploymentCorrelation(analytics.Versions)
    
    // Add performance insights
    performanceInsights := h.generatePerformanceInsights(analytics)
    
    // Add adoption metrics
    adoptionMetrics := h.calculateAdoptionMetrics(analytics)
    
    response := VersionAnalyticsResponse{
        Analytics:            analytics,
        DeploymentCorrelation: deploymentCorrelation,
        PerformanceInsights:  performanceInsights,
        AdoptionMetrics:      adoptionMetrics,
        Recommendations:      h.generateVersionRecommendations(analytics),
        TimeRange:           params.TimeRange,
        GeneratedAt:         time.Now().UTC(),
    }
    
    span.SetFields(tracer.Fields{
        "analytics.versions":     len(analytics.Versions),
        "analytics.deployments":  analytics.TotalDeployments,
        "analytics.success_rate": analytics.SuccessRate,
        "analytics.adoption":     adoptionMetrics.AdoptionRate,
    })
    
    return e.JSON(http.StatusOK, response)
}
```

## Error Handling Strategy

### Version-Specific Error Types
```go
func handleVersionError(e *core.RequestEvent, err error, message string) error {
    if tunnel.IsValidationError(err) {
        return e.JSON(http.StatusBadRequest, ValidationErrorResponse{
            Error:      message,
            Details:    err.Error(),
            Field:      tunnel.GetValidationField(err),
            Suggestion: tunnel.GetValidationSuggestion(err),
            Code:       "VERSION_VALIDATION_FAILED",
        })
    }
    
    if tunnel.IsFileError(err) {
        return e.JSON(http.StatusUnprocessableEntity, FileErrorResponse{
            Error:      message,
            Details:    err.Error(),
            FileType:   tunnel.GetFileType(err),
            Suggestion: tunnel.GetFileSuggestion(err),
            Code:       "FILE_OPERATION_FAILED",
        })
    }
    
    return handleGenericError(e, err, message)
}

func handleUploadError(e *core.RequestEvent, err error, message string) error {
    if tunnel.IsFileSizeError(err) {
        return e.JSON(http.StatusRequestEntityTooLarge, FileSizeErrorResponse{
            Error:      message,
            Details:    "File size exceeds maximum allowed",
            MaxSize:    tunnel.GetMaxFileSize(err),
            ActualSize: tunnel.GetActualFileSize(err),
            Suggestion: "Reduce file size or split into smaller packages",
            Code:       "FILE_TOO_LARGE",
        })
    }
    
    if tunnel.IsCorruptedFileError(err) {
        return e.JSON(http.StatusBadRequest, CorruptedFileErrorResponse{
            Error:      message,
            Details:    "File appears to be corrupted",
            Suggestion: "Re-upload the file or check source integrity",
            Code:       "FILE_CORRUPTED",
        })
    }
    
    return handleVersionError(e, err, message)
}
```

## Integration with Other Handlers

### With App Handlers
```go
// Versions provide deployment packages for apps
func (h *VersionHandlers) getVersionForDeployment(ctx context.Context, versionID string) (*models.Version, error) {
    span := h.tracer.TraceServiceAction(ctx, "version", "deployment_prep")
    defer span.End()
    
    versionModel, err := models.GetVersion(app, versionID)
    if err != nil {
        span.EndWithError(err)
        return nil, err
    }
    
    // Validate version is ready for deployment
    if !versionModel.HasDeploymentZip() {
        err := fmt.Errorf("version has no deployment package")
        span.EndWithError(err)
        return nil, err
    }
    
    // Verify package exists
    statCmd := tunnel.Command{
        Cmd:     fmt.Sprintf("test -f %s", versionModel.GetDeploymentZipPath()),
        Timeout: 10 * time.Second,
    }
    
    _, err = h.executor.RunCommand(e.Request.Context(), statCmd)
    if err != nil {
        span.EndWithError(err)
        return nil, fmt.Errorf("package file not found: %w", err)
    }
    
    span.SetFields(tracer.Fields{
        "version.id":      versionModel.ID,
        "version.ready":   true,
        "package.size":    versionModel.PackageSize,
    })
    
    return versionModel, nil
}
```

### With Deployment Handlers
```go
// Versions validate deployment prerequisites
func (h *VersionHandlers) validateForDeployment(ctx context.Context, versionID, appID string) (*DeploymentValidation, error) {
    span := h.tracer.TraceServiceAction(ctx, "version", "deployment_validation")
    defer span.End()
    
    versionModel, err := models.GetVersion(app, versionID)
    if err != nil {
        span.EndWithError(err)
        return nil, err
    }
    
    if versionModel.AppID != appID {
        err := fmt.Errorf("version does not belong to specified app")
        span.EndWithError(err)
        return nil, err
    }
    
    validation := &DeploymentValidation{
        VersionID:    versionModel.ID,
        AppID:        appID,
        Valid:        true,
        Issues:       []string{},
        Warnings:     []string{},
        Suggestions:  []string{},
    }
    
    // Check package exists and is valid
    if !versionModel.HasDeploymentZip() {
        validation.Valid = false
        validation.Issues = append(validation.Issues, "No deployment package found")
        validation.Suggestions = append(validation.Suggestions, "Upload deployment package first")
    }
    
    // Check validation status
    if versionModel.ValidationStatus == "failed" {
        validation.Valid = false
        validation.Issues = append(validation.Issues, "Package validation failed")
        validation.Suggestions = append(validation.Suggestions, "Fix package issues and re-upload")
    }
    
    // Check package age
    if time.Since(versionModel.Updated) > 30*24*time.Hour {
        validation.Warnings = append(validation.Warnings, "Package is older than 30 days")
        validation.Suggestions = append(validation.Suggestions, "Consider updating package")
    }
    
    span.SetFields(tracer.Fields{
        "validation.valid":     validation.Valid,
        "validation.issues":    len(validation.Issues),
        "validation.warnings":  len(validation.Warnings),
    })
    
    return validation, nil
}
```

## New Dependencies and Injection

### Constructor Function
```go
func NewVersionHandlers(app core.App) (*VersionHandlers, error) {
    // Setup tracing
    tracerFactory := tracer.SetupProductionTracing(os.Stdout)
    serviceTracer := tracerFactory.CreateServiceTracer()
    fileTracer := tracerFactory.CreateFileTracer()
    sshTracer := tracerFactory.CreateSSHTracer()
    
    // Setup tunnel components
    factory := tunnel.NewConnectionFactory(sshTracer)
    poolConfig := tunnel.PoolConfig{
        MaxConnections:     25, // Lower for file operations
        IdleTimeout:       15 * time.Minute,
        HealthCheckInterval: 10 * time.Minute,
    }
    pool := tunnel.NewPool(factory, poolConfig, sshTracer)
    executor := tunnel.NewExecutor(pool, sshTracer)
    
    // Setup file manager
    fileMgr := tunnel.NewFileManager(executor, fileTracer)
    
    // Setup deployment manager for integration
    deployMgr := tunnel.NewDeploymentManager(executor, serviceTracer)
    
    // Setup package validator
    validator := tunnel.NewPackageValidator()
    
    return &VersionHandlers{
        executor:   executor,
        fileMgr:    fileMgr,
        deployMgr:  deployMgr,
        tracer:     serviceTracer,
        fileTracer: fileTracer,
        validator:  validator,
    }, nil
}
```

### File Manager Integration
```go
// Enhanced file operations with tunnel.FileManager
type FileOperationConfig struct {
    MaxFileSize      int64
    MaxTotalSize     int64
    AllowedTypes     []string
    RequiredFiles    []string
    ValidateStructure bool
    CompressionLevel int
    ChecksumType     string
    ProgressTracking bool
}

func (h *VersionHandlers) configureFileOperations() FileOperationConfig {
    return FileOperationConfig{
        MaxFileSize:      104857600, // 100MB
        MaxTotalSize:     157286400, // 150MB
        AllowedTypes:     []string{".zip", ".tar.gz"},
        RequiredFiles:    []string{"pocketbase"},
        ValidateStructure: true,
        CompressionLevel: 9,
        ChecksumType:     "sha256",
        ProgressTracking: true,
    }
}
```

## Step-by-Step Migration Process

### Step 1: Update Handler Structure
```go
// 1. Create new handler struct with dependencies
type VersionHandlers struct {
    executor  tunnel.Executor
    deployMgr tunnel.DeploymentManager
    tracer    tracer.ServiceTracer
    validator tunnel.PackageValidator
}

// 2. Convert functions to methods
func (h *VersionHandlers) createVersion(app core.App, e *core.RequestEvent) error {
    // Implementation with dependencies
}
```

### Step 2: Replace Database Operations
```go
// BEFORE: Direct record manipulation
record := core.NewRecord(collection)
record.Set("app_id", req.AppID)
app.Save(record)

// AFTER: Use models package
versionModel := models.NewVersion()
versionModel.AppID = req.AppID
models.SaveVersion(app, versionModel)
```

### Step 3: Enhance File Operations
```go
// BEFORE: Manual multipart handling
binaryFile, binaryHeader, err := e.Request.FormFile("pocketbase_binary")
publicFiles := e.Request.MultipartForm.File["pb_public_files"]

// AFTER: Use executor for file operations
uploadRequest, err := h.parseUploadRequest(e.Request)
packageResult, err := h.createDeploymentPackage(ctx, packageConfig)
```

### Step 4: Add Comprehensive Tracing
```go
// Add to every operation
span := h.tracer.TraceServiceAction(ctx, "version", operation)
defer span.End()

// File operations
span := h.fileTracer.TraceFileTransfer(ctx, "upload", "deployment_package", fileSize)

// Record metadata
span.SetFields(tracer.Fields{
    "version.id":     versionID,
    "file.size":      fileSize,
    "file.type":      fileType,
})

// Handle errors
if err != nil {
    tracer.RecordError(span, err, "operation failed")
    span.EndWithError(err)
}
```

### Step 5: Implement Package Validation
```go
// Enhanced validation pipeline
func (h *VersionHandlers) validatePackageUpload(ctx context.Context, uploadRequest *tunnel.UploadRequest) (*ValidationResult, error) {
    span := h.fileTracer.TraceFileTransfer(ctx, "validate", "upload", uploadRequest.TotalSize)
    defer span.End()
    
    validation := &ValidationResult{
        Valid:   true,
        Issues:  []string{},
        Warnings: []string{},
    }
    
    // Binary validation
    if binaryFile, exists := uploadRequest.Files["pocketbase_binary"]; exists {
        binaryResult := h.validator.ValidatePocketBaseBinary(binaryFile)
        if !binaryResult.Valid {
            validation.Valid = false
            validation.Issues = append(validation.Issues, binaryResult.Issues...)
        }
    } else {
        validation.Valid = false
        validation.Issues = append(validation.Issues, "PocketBase binary is required")
    }
    
    // Public files validation
    if publicFiles, exists := uploadRequest.Files["pb_public_files"]; exists {
        publicResult := h.validator.ValidatePublicFiles(publicFiles)
        validation.Warnings = append(validation.Warnings, publicResult.Warnings...)
    }
    
    span.SetFields(tracer.Fields{
        "validation.valid":    validation.Valid,
        "validation.issues":   len(validation.Issues),
        "validation.warnings": len(validation.Warnings),
    })
    
    return validation, nil
}
```

## Performance Improvements

### Async File Processing
```go
func (h *VersionHandlers) processVersionPackageAsync(ctx context.Context, versionModel *models.Version, uploadRequest *tunnel.UploadRequest) {
    span := h.fileTracer.TraceFileTransfer(ctx, "process", "deployment_package", uploadRequest.TotalSize)
    defer span.End()
    
    // Create package in background
    packageConfig := tunnel.PackageConfig{
        VersionID:     versionModel.ID,
        Files:         uploadRequest.Files,
        Compression:   tunnel.CompressionBest,
        Validation:    true,
        Checksums:     true,
        Optimization:  true,
    }
    
    progressChan := make(chan tunnel.PackageProgress, 20)
    
    // Monitor progress
    go h.monitorPackageProgress(ctx, versionModel.ID, progressChan)
    
    // Create package using executor
    createCmd := tunnel.Command{
        Cmd:     fmt.Sprintf("cd %s && zip -r %s *", packageConfig.SourceDir, packageConfig.OutputPath),
        Timeout: 10 * time.Minute,
    }
    
    result, err := h.executor.RunCommand(ctx, createCmd)
    
    // Update version model
    if err != nil {
        versionModel.ValidationStatus = "failed"
        versionModel.ValidationError = err.Error()
        span.EndWithError(err)
    } else {
        versionModel.DeploymentZip = result.Filename
        versionModel.PackageSize = result.Size
        versionModel.Checksum = result.Checksum
        versionModel.ValidationStatus = "validated"
        span.Event("package_created")
    }
    
    models.SaveVersion(app, versionModel)
    h.notifyPackageComplete(versionModel.ID, err == nil)
}
```

### Streaming Downloads
```go
func (h *VersionHandlers) streamVersionDownload(app core.App, e *core.RequestEvent) error {
    span := h.fileTracer.TraceFileTransfer(e.Request.Context(), "download", "deployment_package", 0)
    defer span.End()
    
    versionModel, err := models.GetVersion(app, versionID)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "Version not found")
    }
    
    // Prepare streaming download
    streamConfig := tunnel.StreamConfig{
        PackagePath:    versionModel.GetDeploymentZipPath(),
        ChunkSize:      1024 * 1024, // 1MB chunks
        BufferSize:     4096,
        EnableGzip:     true,
        ValidateRange:  true,
        TrackProgress:  true,
    }
    
    // Transfer file using executor
    transferConfig := tunnel.FileTransfer{
        LocalPath:  streamConfig.LocalPath,
        RemotePath: streamConfig.RemotePath,
        Direction:  tunnel.TransferDownload,
        Progress:   true,
    }
    
    err = h.executor.TransferFile(e.Request.Context(), transferConfig)
    if err != nil {
        span.EndWithError(err)
        return handleDownloadError(e, err, "File transfer failed")
    }
    
    span.SetFields(tracer.Fields{
        "version.id":       versionModel.ID,
        "download.size":    versionModel.PackageSize,
        "download.client":  getClientIP(e.Request),
        "download.stream":  true,
    })
    
    return nil
}
```



## Validation Checklist

### ✅ Pre-Migration Validation
- [ ] Map all file operations to tunnel.FileManager
- [ ] Identify validation requirements for packages
- [ ] Plan tracing integration for file operations
- [ ] Design error handling for file-specific errors
- [ ] Assess storage and performance requirements

### ✅ Migration Execution
- [ ] Replace direct database operations with models package
- [ ] Integrate tunnel.FileManager for file operations
- [ ] Add comprehensive package validation pipeline
- [ ] Implement structured error handling with file-specific errors
- [ ] Add tracing for all file and version operations
- [ ] Enhance upload/download with progress tracking
- [ ] Add package optimization and comparison features

### ✅ Post-Migration Validation
- [ ] All handlers use dependency injection
- [ ] File operations use tunnel.FileManager
- [ ] Comprehensive validation pipeline active
- [ ] Structured error responses implemented
- [ ] Tracing coverage for all operations
- [ ] Package integrity checks functional
- [ ] Performance improvements measurable
- [ ] Integration with deployment pipeline working


## Success Metrics

### Performance
- File upload speed improvement > 25%
- Package creation time < 30 seconds for 150MB
- Download streaming efficiency > 90%
- Memory usage stable during large uploads

### Reliability
- Package corruption rate < 0.1%
- Upload success rate > 99%
- Validation accuracy > 99.5%
- Zero data loss during operations

### Observability
- 100% file operation tracing
- Complete validation audit trail
- Detailed performance metrics
- File integrity monitoring

## Timeline
- **Day 1-2**: Update handler structure and models integration
- **Day 3-4**: Integrate tunnel.FileManager and enhanced validation
- **Day 5-6**: Add comprehensive tracing and error handling
- **Day 7**: Testing, optimization, and validation