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
		oninput,
		onchange,
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
		oninput?: (event: Event) => void;
		onchange?: (event: Event) => void;
	} = $props();

	// Base input styles - Vercel-inspired clean design
	const baseInputStyles =
		'block w-full rounded-lg border-gray-200 bg-white shadow-sm transition-all duration-200 focus:border-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-0 disabled:cursor-not-allowed disabled:bg-gray-50 disabled:text-gray-400 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-100 dark:focus:border-gray-100 dark:focus:ring-gray-100 dark:disabled:bg-gray-900 dark:disabled:text-gray-500';

	// Checkbox styles - Vercel-style
	const checkboxStyles =
		'h-4 w-4 rounded border-gray-200 bg-white text-gray-900 transition-all duration-200 focus:ring-2 focus:ring-gray-900 focus:ring-offset-0 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-100 dark:focus:ring-gray-100';

	// Label styles - Vercel-style typography
	const baseLabelStyles = 'block text-sm font-medium text-gray-900 dark:text-gray-100';

	// Error state styles - Vercel-style error handling
	const errorInputStyles = errorText
		? 'border-red-300 focus:border-red-500 focus:ring-red-500 bg-red-50 dark:border-red-700 dark:focus:border-red-500 dark:focus:ring-red-500 dark:bg-red-950'
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
				{checked}
				{disabled}
				{readonly}
				class={inputClasses}
				{onchange}
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
				<span class="text-red-500 dark:text-red-400">*</span>
			{/if}
		</label>

		{#if type === 'select'}
			<select
				{id}
				{value}
				{required}
				{disabled}
				class="mt-1 {inputClasses}"
				{onchange}
				{...restProps}
			>
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
				{value}
				{placeholder}
				{required}
				{disabled}
				{readonly}
				{min}
				{max}
				{step}
				class="mt-1 {inputClasses}"
				{oninput}
				{onchange}
				{...restProps}
			/>
		{/if}
	{/if}

	{#if hasError}
		<p class="text-sm text-red-600 dark:text-red-400">{errorText}</p>
	{:else if helperText}
		<p class="text-xs text-gray-500 dark:text-gray-500">{helperText}</p>
	{/if}
</div>
