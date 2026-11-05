import { writable, get } from "svelte/store";

export const tasks = writable([]);
export const sessions = writable([]);
export const currentSessionId = writable(null);
export const chatHistory = writable([]);
export const isChatLoading = writable(false);

export const selectedDate = writable(new Date());

export function getHeaders() {
	return {
		"Content-Type": "application/json",
		"X-Timezone": Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
		"X-Client-Time": new Date().toISOString(),
	};
}
const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:9090/api";

export async function fetchTasks() {
	try {
		const res = await fetch(`${API_BASE}/tasks`, { headers: getHeaders() });
		const data = await res.json();
		tasks.set(data);
	} catch (err) {
		console.error("Failed to fetch tasks:", err);
	}
}

export async function fetchSessions() {
	try {
		const res = await fetch(`${API_BASE}/sessions`, { headers: getHeaders() });
		const data = await res.json();
		sessions.set(data);

		if (data.length > 0) {
			currentSessionId.update((id) => id || data[0].id);
		} else {
			await createNewSession();
		}
	} catch (err) {
		console.error("Failed to fetch sessions:", err);
	}
}

export async function createNewSession() {
	try {
		const res = await fetch(`${API_BASE}/sessions`, {
			method: "POST",
			headers: getHeaders(),
		});
		const data = await res.json();
		sessions.update((s) => [data, ...s]);
		currentSessionId.set(data.id);
		chatHistory.set([]);
	} catch (err) {
		console.error("Failed to create session:", err);
	}
}

export async function fetchSessionMessages(sessionId) {
	if (!sessionId) return;
	try {
		const res = await fetch(`${API_BASE}/sessions/${sessionId}`, { headers: getHeaders() });
		const data = await res.json();
		chatHistory.set(data);
	} catch (err) {
		console.error("Failed to fetch messages:", err);
	}
}

// Holds the active AbortController so the stop button can cancel the request.
export const currentAbortController = writable(null);

export function cancelChatMessage() {
	const ctrl = get(currentAbortController);
	if (ctrl) {
		ctrl.abort();
		currentAbortController.set(null);
	}
	isChatLoading.set(false);
}

export async function sendChatMessage(sessionId, message) {
	if (!sessionId) return;

	const userMsgId = Date.now() + Math.random();
	chatHistory.update((history) => [
		...history,
		{ id: userMsgId, role: "user", content: message },
	]);
	isChatLoading.set(true);

	const ctrl = new AbortController();
	currentAbortController.set(ctrl);

	try {
		const res = await fetch(`${API_BASE}/chat/${sessionId}`, {
			method: "POST",
			headers: getHeaders(),
			body: JSON.stringify({ message }),
			signal: ctrl.signal,
		});
		const data = await res.json();

		chatHistory.update((history) => [
			...history,
			{
				id: Date.now() + Math.random(),
				role: "assistant",
				content: data.response,
			},
		]);
	} catch (err) {
		if (err.name === "AbortError") {
			chatHistory.update((history) => [
				...history,
				{
					id: Date.now() + Math.random(),
					role: "assistant",
					content: "Response stopped.",
				},
			]);
		} else {
			console.error("Chat failed:", err);
			chatHistory.update((history) => [
				...history,
				{
					id: Date.now() + Math.random(),
					role: "assistant",
					content: "Error: Connection failed.",
				},
			]);
		}
	} finally {
		isChatLoading.set(false);
		currentAbortController.set(null);
		await fetchTasks();
	}
}

export async function toggleTask(task) {
	try {
		const res = await fetch(`${API_BASE}/tasks/${task.id}`, {
			method: "PUT",
			headers: getHeaders(),
			body: JSON.stringify({
				...task,
				is_completed: !task.is_completed,
			}),
		});
		if (res.ok) fetchTasks();
	} catch (err) {
		console.error("Toggle failed:", err);
	}
}

export async function deleteTask(id) {
	try {
		const res = await fetch(`${API_BASE}/tasks/${id}`, {
			method: "DELETE",
			headers: getHeaders(),
		});
		if (res.ok) fetchTasks();
	} catch (err) {
		console.error("Delete failed:", err);
	}
}

export async function deleteSession(id) {
	try {
		const res = await fetch(`${API_BASE}/sessions/${id}`, {
			method: "DELETE",
			headers: getHeaders(),
		});
		if (res.ok) {
			await fetchSessions();
			currentSessionId.update((currentId) => {
				if (currentId === id) {
					const currentSessions = get(sessions);
					return currentSessions.length > 0 ? currentSessions[0].id : null;
				}
				return currentId;
			});
		}
	} catch (err) {
		console.error("Delete session failed:", err);
	}
}
