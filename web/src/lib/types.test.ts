import { describe, expect, it } from 'vitest';
import { ApiError, readApiError } from './types';
import { maskCode, parseBulkPreview } from './bulkParse';

describe('readApiError', () => {
	it('prefers message then msg then string error', async () => {
		const cases: { body: unknown; want: string }[] = [
			{ body: { message: 'from message' }, want: 'from message' },
			{ body: { msg: 'from msg' }, want: 'from msg' },
			{ body: { error: 'from error' }, want: 'from error' },
			{ body: { error: true, msg: 'fiber style' }, want: 'fiber style' },
			{ body: { error: true }, want: '{"error":true}' }
		];

		for (const tc of cases) {
			const res = new Response(JSON.stringify(tc.body), {
				status: 400,
				statusText: 'Bad Request'
			});
			expect(await readApiError(res)).toBe(tc.want);
		}
	});

	it('falls back to plain text', async () => {
		const res = new Response('plain failure', { status: 500, statusText: 'Error' });
		expect(await readApiError(res)).toBe('plain failure');
	});
});

describe('ApiError', () => {
	it('stores status', () => {
		const err = new ApiError(409, 'taken');
		expect(err.status).toBe(409);
		expect(err.message).toBe('taken');
		expect(err.name).toBe('ApiError');
	});
});

describe('parseBulkPreview', () => {
	it('parses tab and comma lines and skips junk', () => {
		const rows = parseBulkPreview('Game A\tKEY-1\nBroken\nGame B,KEY-2\n\n,onlycode\n');
		expect(rows).toEqual([
			{ title: 'Game A', code: 'KEY-1' },
			{ title: 'Game B', code: 'KEY-2' }
		]);
	});
});

describe('maskCode', () => {
	it('masks and reveals', () => {
		expect(maskCode('ABCDEFGHIJ', false)).toBe('ABCD…GHIJ');
		expect(maskCode('SHORT', false)).toBe('••••••••');
		expect(maskCode('ABCDEFGHIJ', true)).toBe('ABCDEFGHIJ');
	});
});
