# CalenAI Agent Reliability Roadmap

## ✅ 1. Tool Recursion / Infinite Loop Protection
Status: DONE

Problem:
- Agent could enter infinite tool-calling loops:
  LLM → Tool → LLM → Tool → ...

Risks:
- Stack overflow
- High CPU usage
- Hanging requests
- No response to user

Implemented:
- Added `maxToolCalls = 10` ceiling limit
- Added `depth` tracking in `callOpenRouterWithTools`
- Returns an explicit error and stops execution when depth exceeds the limit

---

## ✅ 2. Multi-Tool Message Handling Cleanup
Status: DONE

Problem:
- Assistant message was previously duplicated when multiple tool calls were returned in a single response.

Goal:
- Append assistant tool-call message once.
- Append all tool results after it.

Implemented:
- Cleaned up in `callOpenRouterWithTools`: `assistantMessage` with all `ToolCalls` is appended once to `messages`, followed by a loop iterating and appending individual `{ Role: "tool", ToolCallID: tc.ID, Content: res }` responses.
- Eliminates duplicate assistant entries and maintains OpenAI/OpenRouter compliant conversation turns.

---

## ✅ 3. Bulk Task Filtering System
Status: DONE

Problem:
- Deletion and querying previously required specific task IDs. Natural language bulk operations were difficult.

Implemented:
- Implemented `applyTaskFilters` helper supporting:
  - `start_date` and `end_date` (portable SQL date casting for both PostgreSQL and SQLite)
  - `title_contains` (case-insensitive `ILIKE` for PostgreSQL / `LIKE` for SQLite)
  - `completed` boolean flag
- Added tools:
  - `find_tasks`: Searches and filters tasks by title, date range, or completion status.
  - `count_tasks`: Counts matching tasks without loading all rows into memory.
  - `delete_tasks`: Performs batch deletion for all records matching criteria.

---

## 🔄 4. Safe Deletion Confirmation Workflow
Status: IN PROGRESS / PROMPT-ENFORCED

Problem:
- Destructive mass deletion can cause accidental data loss if invoked by ambiguous prompts.

Current State:
- System prompt instructs the model to inspect matching tasks using `find_tasks` or `count_tasks` first and ask for user clarification when ambiguous before deleting.
- Deletions are executed via `delete_task` (single) or `delete_tasks` (bulk).

Planned Enhancement:
- Add a dedicated two-step confirmation state in chat session storage requiring an explicit confirmation token before executing batch deletion.

---

## ✅ 5. Advanced Task Search
Status: DONE

Goal:
- Support natural language task lookup (e.g., "What tasks do I have tomorrow?", "Show GRE tasks", "When is my dentist appointment?").

Implemented:
- Implemented `find_tasks` tool with flexible argument matching.
- Updated system prompt policy: The AI is instructed to ALWAYS call `find_tasks` first when the user refers to existing tasks by title or context rather than dumping all tasks with `get_tasks`.

---

## ✅ 6. Batch Creation Optimization (`add_tasks`)
Status: DONE

Problem:
- Adding recurring tasks (e.g., daily standup for 30 days) required 30 recursive tool call iterations, hitting depth limits and causing high latency.

Implemented:
- Implemented `add_tasks` function calling tool accepting a batch array of tasks.
- Single loop execution in backend with overlap checking and duplicate detection.
- Supported `force: true` parameter for overriding overlap warnings when the user confirms keeping both events.

---

## 🔄 7. User Ownership & Security Layer
Status: FUTURE

Goal:
- Ensure users can only access their own tasks and calendar events in multi-user environments.

Implementation Plan:
- Add `user_id` column to `Task` and `Session` GORM models with foreign key constraints.
- Introduce JWT / session-based user authentication middleware.
- Scope all database queries and tool executions by `user_id`.

---

## 🔄 8. Recovery / Undo System
Status: FUTURE

Goal:
- Allow task restoration after accidental deletion.

Implementation Plan:
- Leverage GORM `DeletedAt` soft-deletes (already supported in model).
- Add `restore_task` / `undo_delete` tool and API endpoint to un-set `deleted_at`.

---

# Progress Summary

- [x] **Tool Recursion / Infinite Loop Protection** *(maxToolCalls = 10)*
- [x] **Multi-Tool Message Handling Cleanup** *(Single assistant message + tool responses)*
- [x] **Bulk Task Filtering System** *(`find_tasks`, `count_tasks`, `delete_tasks`)*
- [x] **Advanced Task Search** *(`find_tasks` with date, title, and status filters)*
- [x] **Batch Creation Optimization** *(`add_tasks` with overlap checks & force flag)*
- [x] **Database Hardening** *(Partial unique index, SQL param injection protection, cloud pooler simple protocol)*
- [ ] **Safe Deletion Confirmation Workflow** *(Interactive multi-step confirmation token)*
- [ ] **User Ownership & Security Layer** *(Auth & multi-tenant user_id isolation)*
- [ ] **Recovery / Undo System** *(Soft-delete restoration tool & endpoint)*
