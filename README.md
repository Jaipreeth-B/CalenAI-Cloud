## Features

-  Natural language scheduling
-  Powered by OpenRouter (Llama 3.1 8B Instruct)
-  Multi-step AI reasoning with tool calling
-  Intelligent calendar & task management
-  Persistent chat history
-  Smart task search, update & deletion
-  SQLite backend
-  Fast Go + Gin REST API


##Example ENV
# OpenRouter configuration
OPENROUTER_API_KEY=your_openrouter_api_key
OPENROUTER_MODEL=meta-llama/llama-3.1-8b-instruct
OPENROUTER_BASE_URL=https://openrouter.ai/api/v1/chat/completions

# Server
PORT=9090

# Database
DATABASE_PATH=calenai.db
