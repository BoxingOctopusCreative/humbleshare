<script lang="ts">
	import LegalDoc from '$lib/components/LegalDoc.svelte';
	import { resolve } from '$app/paths';
</script>

<LegalDoc
	title="Privacy policy"
	description="How HumbleShare collects, uses, and stores account and usage data."
	updated="August 11, 2026"
>
	<p>
		HumbleShare is a private, invite-only site for sharing spare Steam keys and Humble gift links
		with a small group of friends. This policy explains what data an instance stores and why.
	</p>
	<p>
		Each operator runs their own copy of HumbleShare. The operator of the instance you use is
		responsible for how that copy is hosted and who can access it.
	</p>

	<h2>What we collect</h2>
	<p>Depending on how you use the site, an instance may store:</p>
	<ul>
		<li>
			<strong>Account data:</strong> email address, password hash (not the plain password), display
			name, optional bio, Steam gamertag, Discord tag, and avatar image URL.
		</li>
		<li>
			<strong>Sign-in data:</strong> session tokens, optional authenticator (2FA) secrets, and
			linked OAuth identities (Google, Discord, or Steam) when you connect them.
		</li>
		<li>
			<strong>Access requests:</strong> email and optional note when you request an invite.
		</li>
		<li>
			<strong>Claim history:</strong> which inventory item you claimed and when, so keys are not
			handed out twice.
		</li>
		<li>
			<strong>Inventory data (admins):</strong> game titles, Steam store metadata, key or gift-link
			values, import source, and admin notes. Keys and gift links are only shown to admins and to
			the person who successfully claims them.
		</li>
	</ul>

	<h2>How we use data</h2>
	<ul>
		<li>To create and secure your account, and to keep you signed in.</li>
		<li>To approve access requests and send invite email.</li>
		<li>To let you claim a key or gift link and record that claim.</li>
		<li>To show profile details you choose to set (name, bio, tags, avatar).</li>
		<li>To let admins manage inventory, users, and access.</li>
	</ul>
	<p>We do not sell your personal data. We do not use it for advertising.</p>

	<h2>Third-party services</h2>
	<p>An instance may talk to:</p>
	<ul>
		<li>
			<strong>Steam:</strong> store search and app details for game metadata; Steam OpenID if you
			link Steam for login.
		</li>
		<li>
			<strong>Google / Discord:</strong> OAuth sign-in if you choose those providers.
		</li>
		<li>
			<strong>Email provider</strong> (for example Mailgun): invite and account-related mail.
		</li>
		<li>
			<strong>Object storage</strong> (for example Cloudflare R2): optional avatar uploads.
		</li>
		<li>
			<strong>Cloudflare Turnstile</strong> (if enabled): bot checks on public forms.
		</li>
		<li>
			<strong>Humble Bundle:</strong> when an admin imports orders or gift links, the admin’s
			Humble session cookie is sent only for that request. It is not stored by HumbleShare.
		</li>
	</ul>
	<p>
		Those services have their own privacy policies. Steam store pages and gift links you open leave
		HumbleShare and follow Steam or Humble’s rules.
	</p>

	<h2>Cookies and sessions</h2>
	<p>
		HumbleShare sets an HTTP-only session cookie after you sign in so the site can recognize you.
		Clearing cookies or using Logout ends that session on this browser.
	</p>

	<h2>Retention</h2>
	<p>
		Account and claim data stay until an admin removes them or the operator deletes the instance
		database. Access requests may be kept for moderation history. Session records expire after the
		configured session lifetime.
	</p>

	<h2>Your choices</h2>
	<ul>
		<li>Update profile fields and password in Settings.</li>
		<li>Enable or disable 2FA, and link or unlink OAuth providers where the UI allows.</li>
		<li>Ask an admin to delete your account if you no longer want access.</li>
	</ul>

	<h2>Security</h2>
	<p>
		Passwords are stored as salted hashes. Sessions use server-side tokens. Still, no system is
		perfect. Protect your password, use 2FA when you can, and only claim keys on devices you trust.
	</p>

	<h2>Children</h2>
	<p>
		HumbleShare is not directed at children under 13. Access is invite-only for people the operator
		chooses to include.
	</p>

	<h2>Changes</h2>
	<p>
		Operators may update this policy when the product or hosting setup changes. The “Last updated”
		date at the top reflects the latest draft for this codebase.
	</p>

	<h2>Contact</h2>
	<p>
		Questions about privacy on a specific instance should go to that instance’s operator or admins.
		If you reached this page from a hosted copy, use the contact method they provide to members.
	</p>

	<p class="pt-2">
		<a href={resolve('/terms')}>Terms of use</a>
		<span class="text-[var(--color-muted)]"> · </span>
		<a href={resolve('/sitemap')}>Sitemap</a>
	</p>
</LegalDoc>
