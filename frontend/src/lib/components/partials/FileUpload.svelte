<script lang="ts">
	let {
		id,
		label,
		accept = '',
		multiple = false,
		directory = false,
		maxSize = 50 * 1024 * 1024, // 50MB default
		required = false,
		disabled = false,
		helperText,
		errorText,
		value = null,
		class: className = '',
		onFileSelect,
		onError
	}: {
		id: string;
		label: string;
		accept?: string;
		multiple?: boolean;
		directory?: boolean;
		maxSize?: number;
		required?: boolean;
		disabled?: boolean;
		helperText?: string;
		errorText?: string;
		value?: File | File[] | null;
		class?: string;
		onFileSelect?: (files: File | File[] | null) => void;
		onError?: (error: string) => void;
	} = $props();

	let fileInput: HTMLInputElement;
	let dragOver = $state(false);

	// Format file size for display
	function formatFileSize(bytes: number): string {
		if (bytes === 0) return '0 Bytes';
		const k = 1024;
		const sizes = ['Bytes', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	// Validate file
	function validateFile(file: File): string | null {
		if (maxSize && file.size > maxSize) {
			return `File size (${formatFileSize(file.size)}) exceeds maximum allowed size (${formatFileSize(maxSize)})`;
		}

		if (accept && accept.trim()) {
			const acceptedTypes = accept.split(',').map((type) => type.trim());
			const nameParts = file.name.split('.');
			const fileExtension = nameParts.length > 1 ? '.' + nameParts.pop()?.toLowerCase() : '';
			const mimeType = file.type.toLowerCase();

			const isAccepted = acceptedTypes.some((acceptedType) => {
				if (acceptedType.startsWith('.')) {
					return fileExtension === acceptedType.toLowerCase();
				} else if (acceptedType.includes('*')) {
					const baseType = acceptedType.split('/')[0];
					return mimeType.startsWith(baseType);
				} else {
					return mimeType === acceptedType.toLowerCase();
				}
			});

			if (!isAccepted) {
				return `File type not accepted. Accepted types: ${accept}`;
			}
		}

		return null;
	}

	// Handle file selection
	function handleFiles(files: FileList) {
		if (disabled) return;

		const fileArray = Array.from(files);

		// Validate files
		for (const file of fileArray) {
			const error = validateFile(file);
			if (error) {
				onError?.(error);
				return;
			}
		}

		// Notify parent of file selection
		if (multiple || directory) {
			onFileSelect?.(fileArray);
		} else {
			const singleFile = fileArray[0] || null;
			onFileSelect?.(singleFile);
		}
	}

	// Handle file input change
	function handleInputChange(event: Event) {
		const target = event.target as HTMLInputElement;
		if (target.files) {
			handleFiles(target.files);
		}
	}

	// Handle drag events
	function handleDragOver(event: DragEvent) {
		event.preventDefault();
		if (!disabled) {
			dragOver = true;
		}
	}

	function handleDragLeave(event: DragEvent) {
		event.preventDefault();
		// Only set dragOver to false if we're leaving the drop zone entirely
		const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
		const x = event.clientX;
		const y = event.clientY;

		if (x < rect.left || x > rect.right || y < rect.top || y > rect.bottom) {
			dragOver = false;
		}
	}

	function handleDrop(event: DragEvent) {
		event.preventDefault();
		dragOver = false;

		if (disabled) return;

		const files = event.dataTransfer?.files;
		if (files) {
			handleFiles(files);
		}
	}

	// Trigger file input click
	function triggerFileInput() {
		if (!disabled) {
			fileInput.click();
		}
	}

	// Remove file
	function removeFile() {
		// Reset file input
		if (fileInput) {
			fileInput.value = '';
		}

		// Notify parent of change
		if (multiple || directory) {
			onFileSelect?.([] as File[]);
		} else {
			onFileSelect?.(null);
		}
	}

	// Get selected files for display
	let selectedFiles = $derived(value ? (Array.isArray(value) ? value : [value]) : []);

	// Component styles
	const baseStyles =
		'relative block w-full rounded-lg border-2 border-dashed transition-all duration-200';
	const normalStyles =
		'border-gray-300 bg-gray-50 hover:border-gray-400 hover:bg-gray-100 dark:border-gray-700 dark:bg-gray-900 dark:hover:border-gray-600 dark:hover:bg-gray-800';
	const dragOverStyles = 'border-blue-400 bg-blue-50 dark:border-blue-500 dark:bg-blue-950';
	const disabledStyles =
		'border-gray-200 bg-gray-100 cursor-not-allowed dark:border-gray-800 dark:bg-gray-950';
	const errorStyles = 'border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950';

	let dropZoneClasses = $derived(
		`${baseStyles} ${
			disabled ? disabledStyles : errorText ? errorStyles : dragOver ? dragOverStyles : normalStyles
		}`
	);
</script>

<div class="space-y-2 {className}">
	<label for={id} class="block text-sm font-medium text-gray-900 dark:text-gray-100">
		{label}
		{#if required}
			<span class="text-red-500 dark:text-red-400">*</span>
		{/if}
	</label>

	<!-- Hidden file input -->
	<input
		bind:this={fileInput}
		{id}
		type="file"
		{accept}
		multiple={multiple || directory}
		webkitdirectory={directory}
		{disabled}
		{required}
		class="hidden"
		onchange={handleInputChange}
	/>

	<!-- Drop zone -->
	<div
		class={dropZoneClasses}
		ondragover={handleDragOver}
		ondragleave={handleDragLeave}
		ondrop={handleDrop}
		onclick={triggerFileInput}
		role="button"
		tabindex={disabled ? -1 : 0}
		onkeydown={(e) => {
			if ((e.key === 'Enter' || e.key === ' ') && !disabled) {
				e.preventDefault();
				triggerFileInput();
			}
		}}
	>
		<div class="flex flex-col items-center justify-center px-6 py-8 text-center">
			{#if selectedFiles.length > 0}
				<!-- File(s) selected -->
				<div class="space-y-2">
					<div class="text-green-600 dark:text-green-400">
						<svg class="mx-auto h-8 w-8" fill="currentColor" viewBox="0 0 20 20">
							<path
								fill-rule="evenodd"
								d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
								clip-rule="evenodd"
							/>
						</svg>
					</div>

					{#if directory && selectedFiles.length > 0}
						<!-- Directory upload summary -->
						<div class="rounded-md bg-white p-3 shadow-sm dark:bg-gray-800">
							<div class="flex items-center justify-between">
								<div class="flex items-center space-x-3">
									<div class="text-blue-600 dark:text-blue-400">
										<svg class="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
											<path
												d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z"
											/>
										</svg>
									</div>
									<div class="text-left">
										<p class="text-sm font-medium text-gray-900 dark:text-gray-100">
											{selectedFiles.length} files selected
										</p>
										<p class="text-xs text-gray-500 dark:text-gray-400">
											Total: {formatFileSize(selectedFiles.reduce((sum, f) => sum + f.size, 0))}
										</p>
									</div>
								</div>
								<button
									type="button"
									onclick={removeFile}
									class="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
									{disabled}
									aria-label="Remove files"
								>
									<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
										<path
											fill-rule="evenodd"
											d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
											clip-rule="evenodd"
										/>
									</svg>
								</button>
							</div>
						</div>
					{:else}
						<!-- Individual file display -->
						{#each selectedFiles as file (file.name + file.size)}
							<div
								class="flex items-center justify-between rounded-md bg-white p-3 shadow-sm dark:bg-gray-800"
							>
								<div class="flex items-center space-x-3">
									<div class="text-gray-600 dark:text-gray-400">
										<svg class="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
											<path
												fill-rule="evenodd"
												d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z"
												clip-rule="evenodd"
											/>
										</svg>
									</div>
									<div class="text-left">
										<p class="text-sm font-medium text-gray-900 dark:text-gray-100">{file.name}</p>
										<p class="text-xs text-gray-500 dark:text-gray-400">
											{formatFileSize(file.size)}
										</p>
									</div>
								</div>

								<button
									type="button"
									onclick={removeFile}
									class="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
									{disabled}
									aria-label="Remove file"
								>
									<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
										<path
											fill-rule="evenodd"
											d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
											clip-rule="evenodd"
										/>
									</svg>
								</button>
							</div>
						{/each}
					{/if}

					<p class="text-xs text-gray-600 dark:text-gray-400">
						Click to replace or drag new {directory ? 'folder' : multiple ? 'files' : 'file'}
					</p>
				</div>
			{:else}
				<!-- No file selected -->
				<div class="text-gray-400 dark:text-gray-500">
					<svg class="mx-auto h-12 w-12" stroke="currentColor" fill="none" viewBox="0 0 48 48">
						<path
							d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						/>
					</svg>
				</div>

				<div class="mt-4 space-y-2">
					<p class="text-sm font-medium text-gray-900 dark:text-gray-100">
						{dragOver
							? directory
								? 'Drop folder here'
								: 'Drop file here'
							: directory
								? 'Click to select folder or drag and drop'
								: 'Click to upload or drag and drop'}
					</p>

					{#if directory}
						<p class="text-xs text-blue-600 dark:text-blue-400">
							Folder upload mode - all files and subdirectories will be included
						</p>
					{/if}

					{#if accept}
						<p class="text-xs text-gray-500 dark:text-gray-400">
							Accepted formats: {accept}
						</p>
					{/if}

					{#if maxSize}
						<p class="text-xs text-gray-500 dark:text-gray-400">
							Maximum {directory ? 'total ' : ''}size: {formatFileSize(maxSize)}
						</p>
					{/if}
				</div>
			{/if}
		</div>
	</div>

	{#if errorText}
		<p class="text-sm text-red-600 dark:text-red-400">{errorText}</p>
	{:else if helperText}
		<p class="text-xs text-gray-500 dark:text-gray-400">{helperText}</p>
	{/if}
</div>
