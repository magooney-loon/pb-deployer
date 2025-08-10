<script lang="ts">
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
		class: className = '',
		inputClass = '',
		labelClass = '',
		...restProps
	}: {
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
			| 'checkbox';
		placeholder?: string;
		required?: boolean;
		disabled?: boolean;
		readonly?: boolean;
		helperText?: string;
		errorText?: string;
		value?: string | number;
		checked?: boolean;
		options?: Array<{ value: string | number; label: string; disabled?: boolean }>;
		min?: number;
		max?: number;
		step?: number | string;
		class?: string;
		inputClass?: string;
		labelClass?: string;
	} = $props();

	// Base input styles
	const baseInputStyles =
		'block w-full rounded-md border-gray-300 shadow-sm transition-colors focus:border-blue-500 focus:ring-blue-500 disabled:cursor-not-allowed disabled:bg-gray-50 disabled:text-gray-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:disabled:bg-gray-800';

	// Checkbox styles
	const checkboxStyles =
		'h-4 w-4 rounded border-gray-300 text-blue-600 transition-colors focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700';

	// Label styles
	const baseLabelStyles = 'block text-sm font-medium text-gray-700 dark:text-gray-300';

	// Error state styles
	const errorInputStyles = errorText
		? 'border-red-300 focus:border-red-500 focus:ring-red-500 dark:border-red-600'
		: '';

	let inputClasses = $derived(
		type === 'checkbox'
			? `${checkboxStyles} ${inputClass}`
			: `${baseInputStyles} ${errorText ? errorInputStyles : ''} ${inputClass}`
	);

	let labelClasses = $derived(`${baseLabelStyles} ${labelClass}`);
	let hasError = $derived(!!errorText);
</script>

<div class="space-y-1 {className}">
	{#if type === 'checkbox'}
		<div class="flex items-center">
			<input
				{id}
				type="checkbox"
				bind:checked
				{disabled}
				{readonly}
				class={inputClasses}
				{...restProps}
			/>
			<label for={id} class="ml-2 {labelClasses}">
				{label}
				{#if required}
					<span class="text-red-500">*</span>
				{/if}
			</label>
		</div>
	{:else}
		<label for={id} class={labelClasses}>
			{label}
			{#if required}
				<span class="text-red-500">*</span>
			{/if}
		</label>

		{#if type === 'select'}
			<select {id} bind:value {required} {disabled} class="mt-1 {inputClasses}" {...restProps}>
				{#if placeholder}
					<option value="" disabled selected>{placeholder}</option>
				{/if}
				{#each options as option (option.value)}
					<option value={option.value} disabled={option.disabled}>
						{option.label}
					</option>
				{/each}
			</select>
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
				class="mt-1 {inputClasses}"
				{...restProps}
			/>
		{/if}
	{/if}

	{#if hasError}
		<p class="text-sm text-red-600 dark:text-red-400">{errorText}</p>
	{:else if helperText}
		<p class="text-xs text-gray-500 dark:text-gray-400">{helperText}</p>
	{/if}
</div>
