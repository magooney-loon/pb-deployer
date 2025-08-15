<script lang="ts">
	interface Option {
		value: string | number;
		label: string;
		disabled?: boolean;
	}

	interface Props {
		id: string;
		label: string;
		type?:
			| 'text'
			| 'email'
			| 'password'
			| 'number'
			| 'tel'
			| 'url'
			| 'search'
			| 'select'
			| 'checkbox'
			| 'textarea';
		placeholder?: string;
		required?: boolean;
		disabled?: boolean;
		readonly?: boolean;
		helperText?: string;
		errorText?: string;
		value?: string | number;
		checked?: boolean;
		options?: Option[];
		min?: number;
		max?: number;
		step?: number | string;
		rows?: number;
		class?: string;
		oninput?: (event: Event) => void;
		onchange?: (event: Event) => void;
	}

	let {
		id,
		label,
		type = 'text',
		placeholder,
		required = false,
		disabled = false,
		readonly = false,
		helperText,
		errorText,
		value = $bindable(''),
		checked = $bindable(false),
		options = [],
		min,
		max,
		step,
		rows = 3,
		class: className = '',
		oninput,
		onchange,
		...restProps
	}: Props = $props();

	const isError = $derived(!!errorText);
	const hasValue = $derived(value !== '' && value !== null && value !== undefined);
	const isCheckbox = $derived(type === 'checkbox');
	const isSelect = $derived(type === 'select');
	const isTextarea = $derived(type === 'textarea');
</script>

<div class="form-field {className}">
	{#if isCheckbox}
		<!-- Checkbox Layout -->
		<div class="checkbox-container" class:error={isError}>
			<input
				{id}
				type="checkbox"
				bind:checked
				{disabled}
				{readonly}
				class="checkbox"
				class:error={isError}
				{onchange}
				autocomplete="off"
				data-form-type="other"
				{...restProps}
			/>
			<div class="checkbox-content">
				<label for={id} class="checkbox-label">
					{label}
					{#if required}<span class="required">*</span>{/if}
				</label>
				{#if helperText}
					<p class="helper-text">{helperText}</p>
				{/if}
				{#if isError}
					<div class="error-message">
						<svg class="error-icon" fill="currentColor" viewBox="0 0 20 20">
							<path
								fill-rule="evenodd"
								d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
								clip-rule="evenodd"
							/>
						</svg>
						<span>{errorText}</span>
					</div>
				{/if}
			</div>
		</div>
	{:else}
		<!-- Standard Field Layout -->
		<div class="field-container">
			<label for={id} class="field-label">
				{label}
				{#if required}<span class="required">*</span>{/if}
			</label>

			<div class="input-wrapper">
				{#if isSelect}
					<select
						{id}
						bind:value
						{required}
						{disabled}
						class="input select"
						class:error={isError}
						{onchange}
						autocomplete="off"
						data-form-type="other"
						{...restProps}
					>
						{#if placeholder}
							<option value="" disabled class="placeholder-option">
								{placeholder}
							</option>
						{/if}
						{#each options as option (option.value)}
							<option value={option.value} disabled={option.disabled}>
								{option.label}
							</option>
						{/each}
					</select>
				{:else if isTextarea}
					<textarea
						{id}
						bind:value
						{placeholder}
						{required}
						{disabled}
						{readonly}
						{rows}
						class="input textarea"
						class:error={isError}
						{oninput}
						{onchange}
						autocomplete="off"
						autocapitalize="off"
						spellcheck="false"
						data-form-type="other"
						{...restProps}
					></textarea>
				{:else}
					<input
						{id}
						{type}
						bind:value
						{placeholder}
						{required}
						{disabled}
						{readonly}
						{min}
						{max}
						{step}
						class="input"
						class:error={isError}
						{oninput}
						{onchange}
						autocomplete="off"
						autocorrect="off"
						autocapitalize="off"
						spellcheck="false"
						data-form-type="other"
						{...restProps}
					/>
				{/if}

				<!-- Success indicator -->
				{#if hasValue && !isError && !isSelect}
					<div class="success-indicator"></div>
				{/if}
			</div>

			<!-- Helper text -->
			{#if helperText && !isError}
				<p class="helper-text">{helperText}</p>
			{/if}

			<!-- Error message -->
			{#if isError}
				<div class="error-message">
					<svg class="error-icon" fill="currentColor" viewBox="0 0 20 20">
						<path
							fill-rule="evenodd"
							d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
							clip-rule="evenodd"
						/>
					</svg>
					<span>{errorText}</span>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	/* CSS Custom Properties for theming */
	.form-field {
		--color-primary: #3b82f6;
		--color-primary-dark: #2563eb;
		--color-success: #10b981;
		--color-error: #ef4444;
		--color-text: #111827;
		--color-text-secondary: #6b7280;
		--color-bg: #ffffff;
		--color-bg-secondary: #f9fafb;
		--color-border: #d1d5db;
		--color-border-hover: #9ca3af;
		--color-border-focus: var(--color-primary);
		--color-disabled: #9ca3af;
		--color-disabled-bg: #f3f4f6;
		--shadow-focus: 0 0 0 3px rgba(59, 130, 246, 0.1);
		--border-radius: 8px;
		--spacing-xs: 4px;
		--spacing-sm: 8px;
		--spacing-md: 12px;
		--spacing-lg: 16px;
		width: 100%;
	}

	/* Dark mode variables */
	:global([data-theme='dark']) .form-field {
		--color-text: #f9fafb;
		--color-text-secondary: #9ca3af;
		--color-bg: #111827;
		--color-bg-secondary: #1f2937;
		--color-border: #374151;
		--color-border-hover: #4b5563;
		--color-disabled-bg: #1f2937;
		--shadow-focus: 0 0 0 3px rgba(59, 130, 246, 0.2);
	}

	/* Base styles */
	.form-field * {
		box-sizing: border-box;
	}

	/* Checkbox Layout */
	.checkbox-container {
		display: flex;
		align-items: flex-start;
		gap: var(--spacing-md);
		padding: var(--spacing-lg);
		border: 2px solid var(--color-border);
		border-radius: var(--border-radius);
		background-color: var(--color-bg);
		transition: border-color 0.2s ease;
	}

	.checkbox-container:hover {
		border-color: var(--color-border-hover);
	}

	.checkbox-container.error {
		border-color: var(--color-error);
	}

	.checkbox {
		width: 20px;
		height: 20px;
		margin-top: 2px;
		border: 2px solid var(--color-border);
		border-radius: 4px;
		background-color: var(--color-bg);
		cursor: pointer;
		transition: all 0.2s ease;
		flex-shrink: 0;
		appearance: none;
		background-size: 16px 16px;
		background-position: center;
		background-repeat: no-repeat;
	}

	.checkbox:hover {
		border-color: var(--color-border-hover);
	}

	.checkbox:focus {
		outline: none;
		border-color: var(--color-border-focus);
		box-shadow: var(--shadow-focus);
	}

	.checkbox:checked {
		background-color: var(--color-primary);
		border-color: var(--color-primary);
		background-image: url("data:image/svg+xml,%3csvg viewBox='0 0 20 20' fill='white' xmlns='http://www.w3.org/2000/svg'%3e%3cpath fill-rule='evenodd' d='M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z' clip-rule='evenodd'/%3e%3c/svg%3e");
	}

	.checkbox:disabled {
		cursor: not-allowed;
		opacity: 0.5;
		background-color: var(--color-disabled-bg);
		border-color: var(--color-disabled);
	}

	.checkbox.error {
		border-color: var(--color-error);
	}

	.checkbox-content {
		flex: 1;
		min-width: 0;
	}

	.checkbox-label {
		display: block;
		font-size: 14px;
		font-weight: 500;
		color: var(--color-text);
		cursor: pointer;
		line-height: 1.4;
		margin: 0;
	}

	/* Standard Field Layout */
	.field-container {
		display: flex;
		flex-direction: column;
		gap: var(--spacing-sm);
	}

	.field-label {
		display: block;
		font-size: 14px;
		font-weight: 500;
		color: var(--color-text);
		margin: 0;
	}

	.required {
		color: var(--color-error);
		margin-left: var(--spacing-xs);
	}

	/* Input Wrapper */
	.input-wrapper {
		position: relative;
	}

	/* Base Input Styles */
	.input {
		width: 100%;
		padding: var(--spacing-md);
		font-size: 14px;
		line-height: 1.5;
		color: var(--color-text);
		background-color: var(--color-bg);
		border: 2px solid var(--color-border);
		border-radius: var(--border-radius);
		transition: all 0.2s ease;
		outline: none;
	}

	.input::placeholder {
		color: var(--color-text-secondary);
	}

	.input:hover:not(:disabled) {
		border-color: var(--color-border-hover);
	}

	.input:focus {
		border-color: var(--color-border-focus);
		box-shadow: var(--shadow-focus);
	}

	.input:disabled {
		cursor: not-allowed;
		opacity: 0.6;
		background-color: var(--color-disabled-bg);
		color: var(--color-disabled);
	}

	.input:read-only {
		background-color: var(--color-bg-secondary);
		cursor: default;
	}

	.input.error {
		border-color: var(--color-error);
	}

	.input.error:focus {
		border-color: var(--color-error);
		box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.1);
	}

	/* Select Specific */
	.select {
		appearance: none;
		background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 24 24' stroke-width='1.5' stroke='%236b7280'%3e%3cpath stroke-linecap='round' stroke-linejoin='round' d='M19.5 8.25l-7.5 7.5-7.5-7.5'/%3e%3c/svg%3e");
		background-repeat: no-repeat;
		background-position: right var(--spacing-md) center;
		background-size: 16px 16px;
		padding-right: 40px;
		cursor: pointer;
	}

	.select:disabled {
		cursor: not-allowed;
	}

	.placeholder-option {
		color: var(--color-text-secondary);
	}

	/* Textarea Specific */
	.textarea {
		resize: vertical;
		min-height: 80px;
		font-family: inherit;
	}

	/* Success Indicator */
	.success-indicator {
		position: absolute;
		top: 50%;
		right: var(--spacing-md);
		width: 8px;
		height: 8px;
		background-color: var(--color-success);
		border-radius: 50%;
		transform: translateY(-50%);
	}

	/* Helper Text */
	.helper-text {
		font-size: 12px;
		line-height: 1.4;
		color: var(--color-text-secondary);
		margin: 0;
	}

	/* Error Message */
	.error-message {
		display: flex;
		align-items: flex-start;
		gap: var(--spacing-sm);
		font-size: 12px;
		color: var(--color-error);
		margin: 0;
	}

	.error-icon {
		width: 16px;
		height: 16px;
		flex-shrink: 0;
		margin-top: 1px;
	}

	/* Mobile Improvements */
	@media (max-width: 640px) {
		.input {
			font-size: 16px; /* Prevents zoom on iOS */
		}

		.checkbox-container {
			padding: var(--spacing-md);
		}

		.checkbox {
			width: 24px;
			height: 24px;
		}
	}

	/* High contrast mode support */
	@media (prefers-contrast: high) {
		.input,
		.checkbox {
			border-width: 2px;
		}

		.input:focus,
		.checkbox:focus {
			outline: 2px solid var(--color-border-focus);
			outline-offset: 2px;
		}
	}

	/* Reduced motion support */
	@media (prefers-reduced-motion: reduce) {
		.input,
		.checkbox,
		.checkbox-container {
			transition: none;
		}
	}
</style>
