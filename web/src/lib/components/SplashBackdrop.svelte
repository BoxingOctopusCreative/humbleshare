<script lang="ts">
	import { splashBackdrop } from '$lib/splash/backdrop';
</script>

<canvas
	{@attach splashBackdrop}
	class="pointer-events-none fixed inset-0 z-0 opacity-[0.4]"
	aria-hidden="true"
></canvas>
