<script lang="ts">
	let {
		open,
		file,
		aspect,
		title,
		outputSize,
		maxBytes,
		busy = false,
		oncancel,
		onconfirm
	}: {
		open: boolean;
		file: File | null;
		aspect: number;
		title: string;
		outputSize: { width: number; height: number };
		maxBytes: number;
		busy?: boolean;
		oncancel: () => void;
		onconfirm: (file: File) => void | Promise<void>;
	} = $props();

	let imgEl: HTMLImageElement | null = null;
	let imgNatural = $state({ w: 0, h: 0 });
	let frameSize = $state({ w: 0, h: 0 });
	let scale = $state(1);
	let minScale = $state(1);
	let offsetX = $state(0);
	let offsetY = $state(0);
	let dragging = $state(false);
	let exporting = $state(false);
	let error = $state('');

	let dragStart = { x: 0, y: 0, ox: 0, oy: 0 };
	let viewToken = '';

	const maxScale = $derived(minScale * 3);
	const zoomSlider = $derived(
		maxScale <= minScale ? 0 : ((scale - minScale) / (maxScale - minScale)) * 100
	);
	const ready = $derived(imgNatural.w > 0 && frameSize.w > 0);
	const blocked = $derived(busy || exporting);
	const preserveAlpha = $derived(
		Boolean(file && (file.type === 'image/png' || file.type === 'image/webp'))
	);
	const modalMax = $derived(aspect === 1 ? 'max-w-md' : 'max-w-2xl');
	const baseName = $derived(aspect === 1 ? 'avatar' : 'banner');

	$effect(() => {
		if (!open) return;
		const prev = document.body.style.overflow;
		document.body.style.overflow = 'hidden';
		return () => {
			document.body.style.overflow = prev;
		};
	});

	function initView(token: string) {
		const nw = imgNatural.w;
		const nh = imgNatural.h;
		const fw = frameSize.w;
		const fh = frameSize.h;
		if (nw <= 0 || nh <= 0 || fw <= 0 || fh <= 0) return;

		const nextToken = `${token}:${fw}x${fh}`;
		if (viewToken === nextToken) return;
		viewToken = nextToken;

		const cover = Math.max(fw / nw, fh / nh);
		minScale = cover;
		scale = cover;
		offsetX = (fw - nw * cover) / 2;
		offsetY = (fh - nh * cover) / 2;
	}

	function clampOffsets(s: number, ox: number, oy: number) {
		const dw = imgNatural.w * s;
		const dh = imgNatural.h * s;
		const minX = frameSize.w - dw;
		const minY = frameSize.h - dh;
		return {
			x: Math.min(0, Math.max(minX, ox)),
			y: Math.min(0, Math.max(minY, oy))
		};
	}

	function setScale(next: number, pivotX?: number, pivotY?: number) {
		if (minScale <= 0) return;
		const s = Math.min(maxScale, Math.max(minScale, next));
		const px = pivotX ?? frameSize.w / 2;
		const py = pivotY ?? frameSize.h / 2;
		const imgX = (px - offsetX) / scale;
		const imgY = (py - offsetY) / scale;
		const clamped = clampOffsets(s, px - imgX * s, py - imgY * s);
		scale = s;
		offsetX = clamped.x;
		offsetY = clamped.y;
	}

	function onPointerDown(e: PointerEvent) {
		if (blocked || !ready) return;
		dragging = true;
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
		dragStart = { x: e.clientX, y: e.clientY, ox: offsetX, oy: offsetY };
	}

	function onPointerMove(e: PointerEvent) {
		if (!dragging) return;
		const clamped = clampOffsets(
			scale,
			dragStart.ox + (e.clientX - dragStart.x),
			dragStart.oy + (e.clientY - dragStart.y)
		);
		offsetX = clamped.x;
		offsetY = clamped.y;
	}

	function onPointerUp() {
		dragging = false;
	}

	function onZoomInput(e: Event) {
		const v = Number((e.currentTarget as HTMLInputElement).value);
		setScale(minScale + (v / 100) * (maxScale - minScale));
	}

	function onKey(e: KeyboardEvent) {
		if (e.key === 'Escape' && open && !blocked) oncancel();
	}

	function imageAttach(f: File) {
		return (node: HTMLImageElement) => {
			const url = URL.createObjectURL(f);
			node.src = url;
			imgEl = node;
			error = '';
			viewToken = '';

			const onLoad = () => {
				imgNatural = { w: node.naturalWidth, h: node.naturalHeight };
				initView(url);
			};

			if (node.complete && node.naturalWidth > 0) onLoad();
			else node.addEventListener('load', onLoad);

			return () => {
				node.removeEventListener('load', onLoad);
				URL.revokeObjectURL(url);
				if (imgEl === node) imgEl = null;
				imgNatural = { w: 0, h: 0 };
				viewToken = '';
			};
		};
	}

	const fileAttach = $derived(file ? imageAttach(file) : null);

	const cropFrame = (node: HTMLElement) => {
		const measure = () => {
			const w = node.clientWidth;
			const h = node.clientHeight;
			if (w <= 0 || h <= 0) return;
			if (w === frameSize.w && h === frameSize.h) return;
			frameSize = { w, h };
			if (imgEl?.src) initView(imgEl.src);
		};
		measure();
		const ro = new ResizeObserver(measure);
		ro.observe(node);

		const onWheel = (e: WheelEvent) => {
			if (blocked || !ready) return;
			e.preventDefault();
			const rect = node.getBoundingClientRect();
			const factor = e.deltaY < 0 ? 1.08 : 1 / 1.08;
			setScale(scale * factor, e.clientX - rect.left, e.clientY - rect.top);
		};
		node.addEventListener('wheel', onWheel, { passive: false });

		return () => {
			ro.disconnect();
			node.removeEventListener('wheel', onWheel);
		};
	};

	function canvasToBlob(
		canvas: HTMLCanvasElement,
		type: string,
		quality?: number
	): Promise<Blob | null> {
		return new Promise((resolve) => canvas.toBlob(resolve, type, quality));
	}

	async function exportBlob(canvas: HTMLCanvasElement): Promise<{ blob: Blob; type: string; name: string } | null> {
		if (preserveAlpha) {
			const png = await canvasToBlob(canvas, 'image/png');
			if (png && png.size <= maxBytes) {
				return { blob: png, type: 'image/png', name: `${baseName}.png` };
			}
			let quality = 0.92;
			while (quality >= 0.4) {
				const webp = await canvasToBlob(canvas, 'image/webp', quality);
				if (webp && webp.size <= maxBytes) {
					return { blob: webp, type: 'image/webp', name: `${baseName}.webp` };
				}
				quality -= 0.1;
			}
			return null;
		}

		let quality = 0.9;
		while (quality >= 0.4) {
			const jpeg = await canvasToBlob(canvas, 'image/jpeg', quality);
			if (jpeg && jpeg.size <= maxBytes) {
				return { blob: jpeg, type: 'image/jpeg', name: `${baseName}.jpg` };
			}
			quality -= 0.1;
		}
		return null;
	}

	async function confirm() {
		if (!imgEl || !ready || blocked) return;
		exporting = true;
		error = '';
		try {
			const canvas = document.createElement('canvas');
			canvas.width = outputSize.width;
			canvas.height = outputSize.height;
			const ctx = canvas.getContext('2d', { alpha: true });
			if (!ctx) throw new Error('Canvas unavailable');

			const sx = -offsetX / scale;
			const sy = -offsetY / scale;
			const sw = frameSize.w / scale;
			const sh = frameSize.h / scale;

			ctx.clearRect(0, 0, canvas.width, canvas.height);
			ctx.imageSmoothingEnabled = true;
			ctx.imageSmoothingQuality = 'high';
			ctx.drawImage(imgEl, sx, sy, sw, sh, 0, 0, outputSize.width, outputSize.height);

			const exported = await exportBlob(canvas);
			if (!exported) {
				error = 'Image is still too large after compression. Try a different photo.';
				return;
			}

			await onconfirm(new File([exported.blob], exported.name, { type: exported.type }));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to crop image';
		} finally {
			exporting = false;
		}
	}
</script>

<svelte:window onkeydown={onKey} />

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
		role="dialog"
		aria-modal="true"
		aria-labelledby="image-crop-title"
	>
		<button
			type="button"
			class="absolute inset-0 cursor-default"
			aria-label="Close"
			onclick={() => {
				if (!blocked) oncancel();
			}}
		></button>

		<div
			class="relative w-full {modalMax} rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-5 shadow-xl"
		>
			<h2 id="image-crop-title" class="text-lg font-semibold">{title}</h2>
			<p class="mt-1 text-sm text-[var(--color-muted)]">
				Drag to reposition. Use the slider or scroll wheel to zoom.
				{#if preserveAlpha}
					Transparent PNGs and WebPs keep their alpha.
				{/if}
			</p>

			<div
				class="crop-frame relative mx-auto mt-4 w-full overflow-hidden rounded border border-[var(--color-border)]"
				class:cursor-grab={!dragging && ready && !blocked}
				class:cursor-grabbing={dragging}
				class:crop-checker={preserveAlpha}
				class:bg-[var(--color-bg)]={!preserveAlpha}
				style="aspect-ratio: {aspect}; max-width: {aspect === 1 ? '320px' : '100%'}; touch-action: none;"
				{@attach cropFrame}
				onpointerdown={onPointerDown}
				onpointermove={onPointerMove}
				onpointerup={onPointerUp}
				onpointercancel={onPointerUp}
				role="presentation"
			>
				{#if file && fileAttach}
					<img
						alt=""
						draggable="false"
						class="absolute max-w-none select-none"
						style="width: {imgNatural.w > 0
							? `${imgNatural.w * scale}px`
							: 'auto'}; height: {imgNatural.h > 0
							? `${imgNatural.h * scale}px`
							: 'auto'}; left: {offsetX}px; top: {offsetY}px;"
						{@attach fileAttach}
					/>
				{:else}
					<p class="absolute inset-0 flex items-center justify-center text-sm text-[var(--color-muted)]">
						No image selected
					</p>
				{/if}
			</div>

			<label class="mt-4 block text-sm">
				<span class="text-[var(--color-muted)]">Zoom</span>
				<input
					type="range"
					class="mt-1 w-full accent-[var(--color-accent)]"
					min="0"
					max="100"
					step="0.1"
					value={zoomSlider}
					disabled={!ready || blocked}
					oninput={onZoomInput}
				/>
			</label>

			{#if error}
				<p class="mt-2 text-sm text-[var(--color-danger)]">{error}</p>
			{/if}

			<div class="mt-4 flex justify-end gap-2">
				<button
					type="button"
					class="px-3 py-1.5 text-sm text-[var(--color-muted)]"
					onclick={oncancel}
					disabled={blocked}
				>
					Cancel
				</button>
				<button
					type="button"
					class="rounded bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
					disabled={!ready || blocked}
					onclick={confirm}
				>
					{exporting || busy ? 'Saving…' : 'Confirm'}
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.crop-checker {
		background-color: #1e242e;
		background-image:
			linear-gradient(45deg, #2a3340 25%, transparent 25%),
			linear-gradient(-45deg, #2a3340 25%, transparent 25%),
			linear-gradient(45deg, transparent 75%, #2a3340 75%),
			linear-gradient(-45deg, transparent 75%, #2a3340 75%);
		background-size: 16px 16px;
		background-position:
			0 0,
			0 8px,
			8px -8px,
			-8px 0;
	}
</style>
