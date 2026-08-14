<script lang="ts">
	import LegalDoc from '$lib/components/LegalDoc.svelte';
	import { resolve } from '$app/paths';
</script>

<LegalDoc
	title="Terms of use"
	updated="August 14, 2026"
>
	<p>
		These terms cover use of HumbleShare: a private tool for sharing spare Steam product keys and
		Humble gift links among invited friends. By creating an account, requesting access, or using the
		site, you agree to these terms.
	</p>

	<h2>What HumbleShare is</h2>
	<p>
		HumbleShare is invite-only software. It is not a public store, marketplace, or key reseller. It
		helps a small group track spare keys and gift links so each code is claimed once.
	</p>
	<p>
		Humble Bundle, Steam, IGN, Ziff Davis, and other trademarks belong to their owners. HumbleShare
		is not affiliated with, endorsed by, or sponsored by Humble Bundle Inc., IGN Entertainment Inc.,
		Ziff Davis, LLC, Valve Corporation, or their partners.
	</p>

	<h2>Eligibility and accounts</h2>
	<ul>
		<li>You need an invite (or an admin-created account) to use the private parts of the site.</li>
		<li>Provide accurate information and keep your login details secure.</li>
		<li>Do not share your account. You are responsible for activity under your login.</li>
		<li>Admins may suspend or remove access if these terms are broken.</li>
	</ul>

	<h2>Keys and gift links</h2>
	<ul>
		<li>
			Inventory items are donated for personal use by people the operator invites. They are not for
			sale on this site.
		</li>
		<li>
			When you claim a key or gift link, you get one chance to copy it. Treat it as single-use and
			do not publish it.
		</li>
		<li>
			Redeem keys and gift links only through official Steam or Humble flows. Follow Steam and
			Humble’s terms for redemption and region rules.
		</li>
		<li>
			Codes can expire, already be redeemed, or fail region checks. HumbleShare does not guarantee
			that every code works.
		</li>
		<li>
			If a code is dead, report it through the site when that option is available so others are not
			misled.
		</li>
	</ul>

	<h2>Acceptable use</h2>
	<p>You agree not to:</p>
	<ul>
		<li>Sell, auction, or publicly redistribute keys or gift links obtained here.</li>
		<li>Scrape, abuse, or try to claim more than the site’s normal claim flow allows.</li>
		<li>Probe, disrupt, or bypass security, rate limits, or access controls.</li>
		<li>Upload unlawful, harmful, or abusive content (including avatars or profile text).</li>
		<li>Use the service for anything illegal where you live.</li>
	</ul>

	<h2>Admin and operator duties</h2>
	<p>
		Admins may import inventory (including from Humble with a session cookie they control), edit
		titles and codes, manage users, and approve access. Operators must handle Humble and Steam
		credentials lawfully and only for managing inventory they are entitled to share.
	</p>

	<h2>Availability</h2>
	<p>
		The service is provided as-is. It may be offline for maintenance, bugs, or hosting issues. There
		is no uptime promise.
	</p>

	<h2>Disclaimer</h2>
	<p>
		To the fullest extent allowed by law, HumbleShare and its operators are not liable for lost
		keys, failed redemptions, account bans on third-party platforms, or damages from using or being
		unable to use the site. Steam and Humble remain the authorities for product ownership and
		redemption.
	</p>

	<h2>Changes</h2>
	<p>
		These terms may change when the product changes. Continued use after an update means you accept
		the revised terms. The “Last updated” date shows the latest draft for this codebase.
	</p>

	<h2>Contact</h2>
	<p>
		For questions about a specific instance, contact that instance’s operator or admins through the
		channel they use with members.
	</p>

	<p class="pt-2">
		<a href={resolve('/privacy')}>Privacy policy</a>
		<span class="text-[var(--color-muted)]"> · </span>
		<a href={resolve('/sitemap')}>Sitemap</a>
	</p>
</LegalDoc>
