import { loadPatternIcons, pickIcon, type Icon, type Motion } from '$lib/splash/sprites';
import type { Attachment } from 'svelte/attachments';

type Actor = {
	icon: Icon;
	x: number;
	y: number;
	vy: number;
	scale: number;
	phase: number;
	spin: number;
};

function readAccent(): string {
	const v = getComputedStyle(document.documentElement).getPropertyValue('--color-accent').trim();
	return v || '#cc272d';
}

function targetSize(motion: Motion): number {
	if (motion === 'twinkle') return 7 + Math.random() * 8;
	if (motion === 'pulse') return 22 + Math.random() * 12;
	return 26 + Math.random() * 22;
}

function spawn(icons: Icon[], w: number, h: number, fromTop: boolean): Actor {
	const icon = pickIcon(icons);
	const scale = targetSize(icon.motion) / Math.max(icon.w, icon.h);
	const pw = icon.w * scale;
	const ph = icon.h * scale;
	const spin =
		icon.motion === 'spin' ? 2.4 + Math.random() * 3.6 : icon.motion === 'bob' ? 0.2 : 0.1;
	return {
		icon,
		x: Math.random() * Math.max(1, w - pw),
		y: fromTop ? -ph - Math.random() * h * 0.4 : Math.random() * h,
		vy:
			icon.motion === 'twinkle'
				? 16 + Math.random() * 28
				: icon.motion === 'bob'
					? 36 + Math.random() * 58
					: 24 + Math.random() * 52,
		scale,
		phase: Math.random() * Math.PI * 2,
		spin: Math.random() < 0.5 ? -spin : spin
	};
}

function drawActor(ctx: CanvasRenderingContext2D, a: Actor) {
	const iw = a.icon.w * a.scale;
	const ih = a.icon.h * a.scale;
	const motion = a.icon.motion;

	let sway = Math.sin(a.phase) * 6;
	let bob = 0;
	let rot = Math.sin(a.phase) * 0.08;
	let alpha = 1;
	let size = 1;

	if (motion === 'drift') sway = Math.sin(a.phase) * 16;
	if (motion === 'bob') {
		bob = Math.sin(a.phase * 1.6) * 10;
		rot = Math.sin(a.phase) * 0.18;
	}
	if (motion === 'spin') rot = a.phase * a.spin;
	if (motion === 'pulse') {
		size = 0.88 + Math.sin(a.phase * 2.2) * 0.12;
		alpha = 0.72 + Math.sin(a.phase * 2.2) * 0.28;
	}
	if (motion === 'twinkle') {
		const tw = (Math.sin(a.phase * 3.1) + 1) / 2;
		alpha = 0.2 + tw * 0.8;
		size = 0.75 + tw * 0.45;
	}

	ctx.save();
	ctx.globalAlpha = alpha;
	ctx.translate(a.x + sway + iw / 2, a.y + bob + ih / 2);
	ctx.rotate(rot);
	ctx.scale(size, size);
	ctx.drawImage(a.icon.canvas, -iw / 2, -ih / 2, iw, ih);
	ctx.restore();
}

/** Full-viewport pattern-icon waterfall for splash / auth pages. */
export const splashBackdrop: Attachment<HTMLCanvasElement> = (canvas) => {
	const ctx = canvas.getContext('2d');
	if (!ctx) return;

	const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
	let actors: Actor[] = [];
	let raf = 0;
	let last = performance.now();
	let dpr = Math.min(window.devicePixelRatio || 1, 2);
	let icons: Icon[] = [];
	let cancelled = false;

	const resize = () => {
		dpr = Math.min(window.devicePixelRatio || 1, 2);
		const w = window.innerWidth;
		const h = window.innerHeight;
		canvas.width = Math.floor(w * dpr);
		canvas.height = Math.floor(h * dpr);
		canvas.style.width = `${w}px`;
		canvas.style.height = `${h}px`;
		ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
		ctx.imageSmoothingEnabled = true;
		ctx.imageSmoothingQuality = 'high';

		if (!icons.length) return;
		const count = Math.max(90, Math.min(220, Math.floor((w * h) / 6500)));
		actors = Array.from({ length: count }, () => spawn(icons, w, h, false));
	};

	resize();
	window.addEventListener('resize', resize);

	const paintStatic = () => {
		ctx.clearRect(0, 0, window.innerWidth, window.innerHeight);
		for (const a of actors) drawActor(ctx, a);
	};

	const tick = (now: number) => {
		const dt = Math.min(0.05, (now - last) / 1000);
		last = now;
		const w = window.innerWidth;
		const h = window.innerHeight;

		ctx.clearRect(0, 0, w, h);

		for (const a of actors) {
			a.phase += dt * (a.icon.motion === 'spin' ? 1 : 1.4);
			a.y += a.vy * dt;
			drawActor(ctx, a);

			const ph = a.icon.h * a.scale;
			if (a.y > h + ph + 16) {
				Object.assign(a, spawn(icons, w, h, true));
			}
		}

		raf = requestAnimationFrame(tick);
	};

	void loadPatternIcons(readAccent()).then((loaded) => {
		if (cancelled || !loaded.length) return;
		icons = loaded;
		resize();
		if (reduced) {
			paintStatic();
			return;
		}
		last = performance.now();
		raf = requestAnimationFrame(tick);
	});

	return () => {
		cancelled = true;
		cancelAnimationFrame(raf);
		window.removeEventListener('resize', resize);
	};
};
