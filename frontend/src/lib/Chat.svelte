<script>
	import {
		chatHistory,
		sendChatMessage,
		cancelChatMessage,
		isChatLoading,
		currentSessionId,
		fetchSessionMessages,
	} from "./api.js";
	import { tick } from "svelte";
	import { fade } from "svelte/transition";

	let message = "";
	let chatContainer;

	$: if ($currentSessionId) {
		fetchSessionMessages($currentSessionId);
	}

	async function handleSubmit() {
		if (!message.trim() || $isChatLoading || !$currentSessionId) return;
		const msg = message;
		message = "";
		await tick();
		await sendChatMessage($currentSessionId, msg);
	}

	async function scrollToBottom() {
		await tick();
		if (chatContainer) {
			chatContainer.scrollTo({
				top: chatContainer.scrollHeight,
				behavior: "smooth",
			});
		}
	}

	$: if ($chatHistory.length) {
		scrollToBottom();
	}
</script>

<div class="flex flex-col h-full bg-white relative">
	<!-- Chat Header -->
	<div
		class="px-8 py-4 border-b border-gray-200 flex justify-between items-center bg-white/90 backdrop-blur-sm sticky top-0 z-20"
	>
		<h2 class="text-[13px] font-bold text-gray-800">CalenAI</h2>
		{#if $isChatLoading}
			<div class="flex gap-1.5 items-center">
				<div
					class="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce"
				></div>
				<div
					class="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce [animation-delay:0.2s]"
				></div>
				<div
					class="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce [animation-delay:0.4s]"
				></div>
			</div>
		{/if}
	</div>

	<!-- Chat Messages -->
	<div
		bind:this={chatContainer}
		class="flex-1 overflow-y-auto px-6 py-8 space-y-10 scroll-smooth"
	>
		<div class="max-w-3xl mx-auto space-y-10">
			{#each $chatHistory as item (item.id)}
				<div
					class="flex {item.role === 'user'
						? 'justify-end'
						: 'justify-start'}"
				>
					{#if item.role === "assistant"}
						<div class="flex gap-4 max-w-[85%]">
							<div
								class="w-6 h-6 rounded-full border border-gray-300 flex-shrink-0 flex items-center justify-center mt-1 bg-white"
							>
								<span
									class="text-[10px] font-bold text-gray-600"
									>AI</span
								>
							</div>
							<div
								class="bg-gray-50 border border-gray-100 px-4 py-3 rounded-2xl text-[15px] leading-relaxed text-gray-800 whitespace-pre-line"
							>
								{item.content}
							</div>
						</div>
					{:else}
						<div
							class="max-w-[85%] bg-blue-50 border border-blue-100 px-4 py-3 rounded-2xl"
						>
							<div
								class="text-[15px] leading-relaxed text-gray-900"
							>
								{item.content}
							</div>
						</div>
					{/if}
				</div>
			{/each}

			{#if !$currentSessionId}
				<div
					class="flex flex-col items-center justify-center py-20 text-gray-400"
				>
					<p class="text-sm font-medium">
						Select a session from the sidebar to begin.
					</p>
				</div>
			{/if}
		</div>
	</div>

	<!-- Input Bar -->
	<div
		class="p-6 pb-8 bg-gradient-to-t from-white via-white to-transparent pt-10"
	>
		<div class="max-w-3xl mx-auto">
			<form
				on:submit|preventDefault={handleSubmit}
				class="relative flex items-center"
			>
				<input
					type="text"
					bind:value={message}
					disabled={$isChatLoading || !$currentSessionId}
					placeholder="Message CalenAI..."
					class="w-full py-3.5 pl-5 pr-24 bg-white border border-gray-300 rounded-2xl text-[15px] text-gray-900 focus:outline-none focus:border-gray-500 focus:shadow-sm transition-all placeholder:text-gray-400 disabled:opacity-50 disabled:bg-gray-50"
				/>

				{#if $isChatLoading}
					<!-- Stop button -->
					<button
						type="button"
						on:click={cancelChatMessage}
						class="absolute right-2 flex items-center gap-1.5 px-3 h-8 bg-red-500 text-white rounded-xl text-[12px] font-medium hover:bg-red-600 active:scale-95 transition-all cursor-pointer"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							width="12"
							height="12"
							viewBox="0 0 24 24"
							fill="currentColor"
						>
							<rect x="4" y="4" width="16" height="16" rx="2" />
						</svg>
						Stop
					</button>
				{:else}
					<!-- Send button -->
					<button
						type="submit"
						disabled={!message.trim() || !$currentSessionId}
						class="absolute right-2 w-8 h-8 bg-gray-900 text-white rounded-xl flex items-center justify-center hover:bg-gray-800 active:scale-95 transition-all cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							width="16"
							height="16"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2.5"
							stroke-linecap="round"
							stroke-linejoin="round"
							><path d="m5 12 7-7 7 7" /><path
								d="M12 19V5"
							/></svg
						>
					</button>
				{/if}
			</form>
			<p class="text-[11px] text-center mt-3 text-gray-500">
				CalenAI can make mistakes. Consider verifying important
				information.
			</p>
		</div>
	</div>
</div>

<style>
	.overflow-y-auto::-webkit-scrollbar {
		width: 0px;
	}
</style>
