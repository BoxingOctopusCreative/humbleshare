const PATTERN_URL = '/media/svg/HBPattern.svg';

export type Motion = 'twinkle' | 'pulse' | 'spin' | 'bob' | 'drift' | 'sway';

export type Icon = {
	canvas: HTMLCanvasElement;
	w: number;
	h: number;
	motion: Motion;
};

const ALPHA = 28;
const MIN_AREA = 10;

function classify(w: number, h: number): Motion {
	const m = Math.max(w, h);
	const ar = w / h;
	if (m < 22) return 'twinkle';
	if (h > w * 1.28) return 'bob';
	if (w > h * 1.28) return 'drift';
	if (m >= 50 && ar >= 0.85 && ar <= 1.18) return 'spin';
	if (m < 50 && ar >= 0.75 && ar <= 1.35) return 'pulse';
	return 'sway';
}

function tint(src: HTMLCanvasElement, color: string): HTMLCanvasElement {
	const out = document.createElement('canvas');
	out.width = src.width;
	out.height = src.height;
	const ctx = out.getContext('2d');
	if (!ctx) return src;
	ctx.fillStyle = color;
	ctx.fillRect(0, 0, out.width, out.height);
	ctx.globalCompositeOperation = 'destination-in';
	ctx.drawImage(src, 0, 0);
	return out;
}

function extractFromImage(img: HTMLImageElement, size: number): HTMLCanvasElement[] {
	const sheet = document.createElement('canvas');
	sheet.width = size;
	sheet.height = size;
	const ctx = sheet.getContext('2d', { willReadFrequently: true });
	if (!ctx) return [];
	ctx.drawImage(img, 0, 0, size, size);

	const { data, width, height } = ctx.getImageData(0, 0, size, size);
	const visited = new Uint8Array(width * height);
	const crops: HTMLCanvasElement[] = [];
	const idx = (x: number, y: number) => y * width + x;

	for (let y = 0; y < height; y++) {
		for (let x = 0; x < width; x++) {
			const start = idx(x, y);
			if (visited[start] || data[start * 4 + 3] < ALPHA) continue;

			let minX = x;
			let minY = y;
			let maxX = x;
			let maxY = y;
			let area = 0;
			const stack = [start];
			visited[start] = 1;

			while (stack.length) {
				const p = stack.pop()!;
				area++;
				const px = p % width;
				const py = (p / width) | 0;
				if (px < minX) minX = px;
				if (py < minY) minY = py;
				if (px > maxX) maxX = px;
				if (py > maxY) maxY = py;

				const neighbors = [
					px > 0 ? p - 1 : -1,
					px < width - 1 ? p + 1 : -1,
					py > 0 ? p - width : -1,
					py < height - 1 ? p + width : -1
				];
				for (const n of neighbors) {
					if (n < 0 || visited[n] || data[n * 4 + 3] < ALPHA) continue;
					visited[n] = 1;
					stack.push(n);
				}
			}

			if (area < MIN_AREA) continue;
			const pad = 2;
			const sx = Math.max(0, minX - pad);
			const sy = Math.max(0, minY - pad);
			const sw = Math.min(width, maxX + 1 + pad) - sx;
			const sh = Math.min(height, maxY + 1 + pad) - sy;
			if (sw < 2 || sh < 2) continue;

			const crop = document.createElement('canvas');
			crop.width = sw;
			crop.height = sh;
			const cropCtx = crop.getContext('2d');
			if (!cropCtx) continue;
			cropCtx.drawImage(sheet, sx, sy, sw, sh, 0, 0, sw, sh);
			crops.push(crop);
		}
	}

	return crops;
}

/** Load HBPattern, split ink into icons, tint to the site accent. */
export async function loadPatternIcons(color: string): Promise<Icon[]> {
	const img = new Image();
	img.crossOrigin = 'anonymous';
	img.src = PATTERN_URL;
	await img.decode();

	const size = 1000;
	return extractFromImage(img, size).map((canvas) => {
		const w = canvas.width;
		const h = canvas.height;
		return {
			canvas: tint(canvas, color),
			w,
			h,
			motion: classify(w, h)
		};
	});
}

export function pickIcon(icons: Icon[]): Icon {
	const mains = icons.filter((i) => i.motion !== 'twinkle');
	const twinkles = icons.filter((i) => i.motion === 'twinkle');
	if (mains.length && (twinkles.length === 0 || Math.random() > 0.28)) {
		return mains[Math.floor(Math.random() * mains.length)];
	}
	if (twinkles.length) return twinkles[Math.floor(Math.random() * twinkles.length)];
	return icons[Math.floor(Math.random() * icons.length)];
}
