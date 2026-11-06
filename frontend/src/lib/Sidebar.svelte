<script>
	import {
		sessions,
		currentSessionId,
		createNewSession,
		deleteSession,
	} from "./api.js";
	import { onMount, onDestroy } from "svelte";

	let currentTimeStr = "";
	let timer;

	function updateCurrentTime() {
		const now = new Date();
		currentTimeStr = now.toLocaleTimeString([], {
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit",
			hour12: true,
		});
	}

	onMount(() => {
		updateCurrentTime();
		timer = setInterval(updateCurrentTime, 1000);
	});

	onDestroy(() => {
		if (timer) clearInterval(timer);
	});

	// Reformat session title to the user's local timezone if it's a default "New Chat" title
	function formatSessionTitle(session) {
		if (!session) return "";
		if (session.created_at && (!session.title || session.title.startsWith("New Chat"))) {
			const d = new Date(session.created_at);
			if (!isNaN(d.getTime())) {
				const datePart = d.toLocaleDateString([], {
					month: "short",
					day: "2-digit",
				});
				const timePart = d.toLocaleTimeString([], {
					hour: "2-digit",
					minute: "2-digit",
					hour12: false,
				});
				return `New Chat ${datePart}, ${timePart}`;
			}
		}
		return session.title;
	}
</script>

<div class="flex flex-col h-full bg-gray-50 text-gray-900">
	<div class="p-4 pb-2">
		<button
			on:click={createNewSession}
			class="w-full flex items-center justify-between px-3 py-2.5 bg-white border border-gray-200 rounded-xl hover:border-gray-400 hover:shadow-sm transition-all cursor-pointer group"
		>
			<span class="text-[13px] font-bold text-gray-800"
				>New Chat</span
			>
			<div
				class="w-5 h-5 rounded flex items-center justify-center text-gray-400 group-hover:text-gray-900 transition-all"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					width="14"
					height="14"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2.5"
					stroke-linecap="round"
					stroke-linejoin="round"
					><path d="M5 12h14" /><path
						d="M12 5v14"
					/></svg
				>
			</div>
		</button>
	</div>

	<div class="flex-1 overflow-y-auto px-4 py-2 space-y-1">
		<h3
			class="text-[10px] font-bold text-gray-400 uppercase tracking-widest px-3 mb-3 mt-4"
		>
			History
		</h3>
		{#each $sessions as session (session.id)}
			<div
				class="group relative px-3 py-2.5 rounded-lg cursor-pointer transition-all {$currentSessionId ===
				session.id
					? 'bg-gray-200 text-gray-900 font-semibold'
					: 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'}"
				on:click={() =>
					currentSessionId.set(session.id)}
			>
				<p class="text-[13px] truncate pr-6">
					{formatSessionTitle(session)}
				</p>

				<button
					on:click|stopPropagation={() =>
						deleteSession(session.id)}
					class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 opacity-0 group-hover:opacity-100 hover:text-red-500 transition-all cursor-pointer p-1"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="12"
						height="12"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						><path d="M18 6 6 18" /><path
							d="m6 6 12 12"
						/></svg
					>
				</button>
			</div>
		{/each}
	</div>

	<div class="p-4 border-t border-gray-200">
		<div class="flex items-center gap-3 px-3 py-2">
			<div
				class="w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center text-gray-700 font-bold text-xs border border-gray-300"
			>
				U
			</div>
			<div class="flex-1 min-w-0">
				<p
					class="text-[13px] font-bold text-gray-900 truncate"
				>
					Local User
				</p>
				<p
					class="text-[11px] font-medium text-gray-500 flex items-center gap-1.5"
				>
					<span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
					<span>{currentTimeStr || "Connected"}</span>
				</p>
			</div>
		</div>
	</div>
</div>

<style>
	.overflow-y-auto::-webkit-scrollbar {
		width: 0px;
	}
</style>
