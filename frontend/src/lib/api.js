import { writable, get } from "svelte/store";

export const tasks = writable([]);
export const sessions = writable([]);
export const currentSessionId = writable(null);
export const chatHistory = writable([]);
export const isChatLoading = writable(false);

// New store for the selected calendar date
export const selectedDate = writable(new Date());

const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:9090/api";
console.log(API_BASE);
export async function fetchTasks() {
	try {
		const res = await fetch(`${API_BASE}/tasks`);
		const data = await res.json();
		tasks.set(data);
	} catch (err) {
		console.error("Failed to fetch tasks:", err);
	}
}

export async function fetchSessions() {
	try {
		const res = await fetch(`${API_BASE}/sessions`);
		const data = await res.json();
		sessions.set(data);

		// Auto-select the first session if none is selected, OR create one if empty
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
		const res = await fetch(`${API_BASE}/sessions/${sessionId}`);
		const data = await res.json();
		chatHistory.set(data);
	} catch (err) {
		console.error("Failed to fetch messages:", err);
	}
}

export async function sendChatMessage(sessionId, message) {
	if (!sessionId) return;

	const userMsgId = Date.now() + Math.random();
	chatHistory.update((history) => [
		...history,
		{ id: userMsgId, role: "user", content: message },
	]);
	isChatLoading.set(true);

	try {
		const res = await fetch(`${API_BASE}/chat/${sessionId}`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ message }),
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
		console.error("Chat failed:", err);
		chatHistory.update((history) => [
			...history,
			{
				id: Date.now() + Math.random(),
				role: "assistant",
				content: "Error: Connection failed.",
			},
		]);
	} finally {
		isChatLoading.set(false);
		await fetchTasks();
	}
}

export async function toggleTask(task) {
	try {
		const res = await fetch(`${API_BASE}/tasks/${task.id}`, {
			method: "PUT",
			headers: { "Content-Type": "application/json" },
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
		});
		if (res.ok) {
			await fetchSessions();
			currentSessionId.update((currentId) => {
				if (currentId === id) {
					const currentSessions = get(sessions);
					return currentSessions.length > 0
						? currentSessions[0].id
						: null;
				}
				return currentId;
			});
		}
	} catch (err) {
		console.error("Delete session failed:", err);
	}
}
