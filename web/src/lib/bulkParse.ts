/** Parse Title[TAB]Code or Title,Code lines into preview rows. */
export type BulkPreviewRow = { title: string; code: string };

export function parseBulkPreview(blob: string): BulkPreviewRow[] {
	return blob
		.split(/\r?\n/)
		.map((line) => line.trim())
		.filter(Boolean)
		.map((line) => {
			const parts = line.includes('\t') ? line.split('\t') : line.split(',');
			const title = (parts[0] ?? '').trim();
			const code = (parts.slice(1).join(',').trim() || '').trim();
			return { title, code };
		})
		.filter((r) => r.title && r.code);
}

/** Mask a code for admin screenshare; reveal shows the full value. */
export function maskCode(value: string, revealed: boolean): string {
	if (revealed) return value;
	if (value.length <= 8) return '••••••••';
	return `${value.slice(0, 4)}…${value.slice(-4)}`;
}
