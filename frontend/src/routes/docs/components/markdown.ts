export interface MarkdownOptions {
	baseUrl?: string;
	sanitize?: boolean;
}

export class MarkdownParser {
	private options: MarkdownOptions;

	constructor(options: MarkdownOptions = {}) {
		this.options = {
			baseUrl: '',
			sanitize: true,
			...options
		};
	}

	parse(markdown: string): string {
		let html = this.preprocessMarkdown(markdown);

		// Process in order of precedence to avoid conflicts
		html = this.parseCodeBlocks(html);
		html = this.parseTables(html);
		html = this.parseHeaders(html);
		html = this.parseBlockquotes(html);
		html = this.parseHorizontalRules(html);
		html = this.parseLists(html);
		html = this.parseInlineElements(html);
		html = this.parseParagraphs(html);

		return html.trim();
	}

	private preprocessMarkdown(markdown: string): string {
		return markdown.replace(/\r\n/g, '\n').replace(/\r/g, '\n').trim();
	}

	private parseHeaders(html: string): string {
		return html
			.replace(
				/^### (.*$)/gim,
				'<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-3 mt-6">$1</h3>'
			)
			.replace(
				/^## (.*$)/gim,
				'<h2 class="text-xl font-bold text-gray-900 dark:text-gray-100 mb-4 mt-8">$1</h2>'
			)
			.replace(
				/^# (.*$)/gim,
				'<h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-6 mt-0">$1</h1>'
			);
	}

	private parseCodeBlocks(html: string): string {
		return html.replace(/```(\w+)?\n([\s\S]*?)```/g, (match, lang, code) => {
			const language = lang || '';
			const cleanCode = this.escapeHtml(code.trim());

			return `<div class="my-6 overflow-hidden rounded-lg border border-gray-200 dark:border-gray-800">
				<div class="bg-gray-50 px-4 py-2 text-xs font-mono text-gray-600 dark:bg-gray-900 dark:text-gray-400">
					${language || 'code'}
				</div>
				<pre class="overflow-x-auto bg-black p-4"><code class="text-sm text-gray-100 font-mono leading-relaxed">${cleanCode}</code></pre>
			</div>`;
		});
	}

	private parseInlineElements(html: string): string {
		// Inline code (before links to avoid conflicts)
		html = html.replace(
			/`([^`]+)`/g,
			'<code class="rounded bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 px-2 py-1 text-sm font-mono text-pink-600 dark:text-orange-400">$1</code>'
		);

		// Links with external detection
		html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (match, text, url) => {
			const isExternal = url.startsWith('http') || url.startsWith('//');
			const target = isExternal ? ' target="_blank" rel="noopener noreferrer"' : '';
			const icon = isExternal ? '<span class="ml-1 text-xs opacity-60">↗</span>' : '';

			return `<a href="${url}"${target} class="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 font-medium underline decoration-1 underline-offset-2 transition-colors">${text}${icon}</a>`;
		});

		// Bold and italic
		html = html
			.replace(
				/\*\*\*(.*?)\*\*\*/g,
				'<strong class="font-bold text-gray-900 dark:text-gray-100"><em>$1</em></strong>'
			)
			.replace(
				/\*\*(.*?)\*\*/g,
				'<strong class="font-semibold text-gray-900 dark:text-gray-100">$1</strong>'
			)
			.replace(/\*(.*?)\*/g, '<em class="italic text-gray-800 dark:text-gray-200">$1</em>');

		return html;
	}

	private parseBlockquotes(html: string): string {
		return html.replace(/^> (.*$)/gim, (match, text) => {
			// Enhanced callout detection
			if (text.match(/^\*\*(Tip|Pro Tip)\*\*:/)) {
				return `<div class="my-4 rounded-lg border-l-4 border-green-500 bg-green-50 dark:bg-green-950/20 dark:border-green-400 p-4">
					<p class="text-green-800 dark:text-green-200 font-medium">${text}</p>
				</div>`;
			}

			if (text.match(/^\*\*(Warning|Caution|Important)\*\*:/)) {
				return `<div class="my-4 rounded-lg border-l-4 border-yellow-500 bg-yellow-50 dark:bg-yellow-950/20 dark:border-yellow-400 p-4">
					<p class="text-yellow-800 dark:text-yellow-200 font-medium">${text}</p>
				</div>`;
			}

			if (text.match(/^\*\*(Note|Info|Remember)\*\*:/)) {
				return `<div class="my-4 rounded-lg border-l-4 border-blue-500 bg-blue-50 dark:bg-blue-950/20 dark:border-blue-400 p-4">
					<p class="text-blue-800 dark:text-blue-200 font-medium">${text}</p>
				</div>`;
			}

			// Regular blockquote
			return `<blockquote class="my-4 border-l-4 border-gray-300 dark:border-gray-600 bg-gray-50 dark:bg-gray-800/50 p-4 italic">
				<p class="text-gray-700 dark:text-gray-300">${text}</p>
			</blockquote>`;
		});
	}

	private parseLists(html: string): string {
		// Process lists line by line to handle nesting
		const lines = html.split('\n');
		const result: string[] = [];
		let inList = false;
		let listItems: string[] = [];
		let listType = '';

		for (let i = 0; i < lines.length; i++) {
			const line = lines[i];
			const unorderedMatch = line.match(/^(\s*)[-*+]\s+(.*)$/);
			const orderedMatch = line.match(/^(\s*)\d+\.\s+(.*)$/);

			if (unorderedMatch) {
				if (!inList) {
					inList = true;
					listType = 'ul';
					listItems = [];
				}
				listItems.push(
					`<li class="text-gray-700 dark:text-gray-300 mb-1">${unorderedMatch[2]}</li>`
				);
			} else if (orderedMatch) {
				if (!inList) {
					inList = true;
					listType = 'ol';
					listItems = [];
				}
				listItems.push(`<li class="text-gray-700 dark:text-gray-300 mb-1">${orderedMatch[2]}</li>`);
			} else {
				if (inList) {
					const listClass =
						listType === 'ul'
							? 'list-disc list-inside space-y-1 mb-4 pl-4'
							: 'list-decimal list-inside space-y-1 mb-4 pl-4';

					result.push(`<${listType} class="${listClass}">${listItems.join('')}</${listType}>`);
					inList = false;
					listItems = [];
				}
				result.push(line);
			}
		}

		// Handle list at end of content
		if (inList && listItems.length > 0) {
			const listClass =
				listType === 'ul'
					? 'list-disc list-inside space-y-1 mb-4 pl-4'
					: 'list-decimal list-inside space-y-1 mb-4 pl-4';

			result.push(`<${listType} class="${listClass}">${listItems.join('')}</${listType}>`);
		}

		return result.join('\n');
	}

	private parseTables(html: string): string {
		const lines = html.split('\n');
		const result: string[] = [];
		let inTable = false;
		let tableRows: string[] = [];
		let headerProcessed = false;

		for (let i = 0; i < lines.length; i++) {
			const line = lines[i].trim();

			// Detect table rows (lines with pipes)
			if (line.includes('|') && line.split('|').length >= 3) {
				if (!inTable) {
					inTable = true;
					tableRows = [];
					headerProcessed = false;
				}

				// Skip separator lines (|---|---|)
				if (line.match(/^\|[\s\-|:]+\|$/)) {
					headerProcessed = true;
					continue;
				}

				const cells = line
					.split('|')
					.slice(1, -1) // Remove empty first/last
					.map((cell) => cell.trim());

				if (cells.length > 0) {
					const isHeader = !headerProcessed;
					const cellTag = isHeader ? 'th' : 'td';
					const cellClass = isHeader
						? 'px-4 py-3 bg-gray-50 dark:bg-gray-800 text-left text-sm font-semibold text-gray-900 dark:text-gray-100 border-b border-gray-200 dark:border-gray-700'
						: 'px-4 py-3 text-sm text-gray-700 dark:text-gray-300 border-b border-gray-200 dark:border-gray-700';

					const cellElements = cells
						.map((cell) => `<${cellTag} class="${cellClass}">${cell}</${cellTag}>`)
						.join('');

					const rowClass = isHeader
						? ''
						: 'hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors';
					tableRows.push(`<tr class="${rowClass}">${cellElements}</tr>`);
				}
			} else {
				if (inTable && tableRows.length > 0) {
					result.push(`<div class="my-6 overflow-hidden rounded-lg border border-gray-200 dark:border-gray-800">
						<table class="w-full border-collapse">${tableRows.join('')}</table>
					</div>`);
					inTable = false;
					tableRows = [];
				}
				result.push(line);
			}
		}

		// Handle table at end
		if (inTable && tableRows.length > 0) {
			result.push(`<div class="my-6 overflow-hidden rounded-lg border border-gray-200 dark:border-gray-800">
				<table class="w-full border-collapse">${tableRows.join('')}</table>
			</div>`);
		}

		return result.join('\n');
	}

	private parseHorizontalRules(html: string): string {
		return html.replace(/^---+$/gm, '<hr class="my-8 border-gray-200 dark:border-gray-800">');
	}

	private parseParagraphs(html: string): string {
		// Split by double newlines for paragraphs
		const blocks = html.split(/\n\s*\n/);

		return blocks
			.map((block) => {
				block = block.trim();
				if (!block) return '';

				// Don't wrap if it's already a block element
				if (this.isBlockElement(block)) {
					return block;
				}

				// Convert remaining newlines to <br> and wrap in paragraph
				const content = block.replace(/\n/g, '<br>');
				return `<p class="mb-4 text-gray-700 dark:text-gray-300 leading-relaxed">${content}</p>`;
			})
			.filter((block) => block.length > 0)
			.join('\n\n');
	}

	private isBlockElement(html: string): boolean {
		return /^<(h[1-6]|div|table|ul|ol|blockquote|pre|hr|p)/.test(html.trim());
	}

	private escapeHtml(text: string): string {
		const div = document.createElement('div');
		div.textContent = text;
		return div.innerHTML;
	}
}

// Default parser instance
export const markdownParser = new MarkdownParser();

// Helper function for components
export function parseMarkdown(content: string, options?: MarkdownOptions): string {
	const parser = new MarkdownParser(options);
	return parser.parse(content);
}
