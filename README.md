<p align="center"><img src="https://github.com/JayyDoesDev/airo/blob/main/.github/assets/aira.png" alt="aira" width="400"/></p>
<h1 align="center">Aira</h1>
<h2 align="center">💬✨ Your unhinged AI sidekick — powered by DiscordGo + DeepSeek</h2>

<div>
  <h2 align="center">
    <img src="https://img.shields.io/github/commit-activity/m/jayydoesdev/airo">
    <img src="https://img.shields.io/github/license/jayydoesdev/airo">
    <img src="https://img.shields.io/github/languages/top/jayydoesdev/airo">
    <img src="https://img.shields.io/github/contributors/jayydoesdev/airo">
    <img src="https://img.shields.io/github/last-commit/jayydoesdev/airo">
  </h2>
</div>

---

- 🧠 **AI Chat** — Unhinged, sassy replies inspired by Aira from *Dandadan*. No full sentences required.
- 🔎 **Web Search** — Searches the web via Exa and cites sources in a references embed
- 📊 **Chart Generation** — Renders bar, line, pie, radar, and horizontal bar charts as PNG images
- 🌐 **Server Vibe Check** — Reads recent channel messages and gives a brutally honest vibe report
- 🔊 **Voice Chat** — Joins voice channels and speaks responses aloud via Piper TTS
- 🛡️ **Moderation** — Kick, ban, assign/remove roles, send DMs, list roles
- 🎭 **Status Control** — Sets her own Discord status and activity on command
- 🧵 **Reply Awareness** — Responds to replies on her own messages with full context
- 📝 **Persistent Memory** — Short and long-term memory stored in encrypted msgpack
- 🔀 **Async Task Queue** — All Discord actions run asynchronously and in order
- 🛡️ **Prompt Injection Defense** — Go-layer regex sanitizer + hardened system prompt
- 🤖 **Mention to Activate** — Just mention her or reply to her messages

---

## 📁 Project Structure

```
airo/
├── discord/
│   ├── events/        # Message handler, search, vibe check, permissions, cooldown
│   └── voice/         # Voice connection manager + Piper TTS pipeline
├── lib/               # AI client abstraction, prompts, injection sanitizer
├── skills/
│   ├── actions/       # Action parser, handler, memory store
│   ├── chart.go       # go-charts PNG renderer
│   └── exa.go         # Exa web search client
├── tasks/             # Async task queue
├── main.go
└── .env
```

---

## 🚀 Getting Started

### 1. Clone the repo

```bash
git clone https://github.com/jayydoesdev/airo.git
cd airo
```

### 2. Set up your environment

Create a `.env` file in the root:

```env
DISCORD_BOT_TOKEN=your_bot_token
DEEPSEEK_API_KEY=your_deepseek_api_key
EXA_API_KEY=your_exa_api_key
EXA_RESULT_LIMIT=5

# Optional — only needed for voice chat
PIPER_MODEL=/path/to/en_US-lessac-medium.onnx
PIPER_SAMPLE_RATE=22050
```

> Enable **Message Content Intent**, **Guild Members Intent**, and **Guild Voice States Intent** in the Discord Developer Portal.

### 3. Install system dependencies

```bash
# Required for chart rendering (already pure-Go, no extra deps)

# Required for voice chat only
apt install libopus-dev ffmpeg       # Ubuntu/Debian
brew install opus ffmpeg             # macOS
```

### 4. (Optional) Set up voice chat

Download a [Piper](https://github.com/rhasspy/piper/releases) binary and a voice model:

```bash
# Download model (~60MB)
curl -LO https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx
curl -LO https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json
```

Set `PIPER_MODEL` and `PIPER_SAMPLE_RATE` in your `.env`.

### 5. Run the bot

```bash
go run main.go
```

---

## ⚙️ Actions

| Action | Description |
|---|---|
| `kick_user` | Kicks a member |
| `ban_user` | Bans a member |
| `assign_role` | Gives a user a role |
| `remove_role` | Removes a role from a user |
| `dm_user` | Sends a private message to a user |
| `list_user_roles` | Lists roles for a specific member |
| `generate_chart` | Renders a chart as a PNG embed |
| `set_status` | Sets the bot's Discord status and activity |
| `join_voice` | Joins the user's current voice channel |
| `leave_voice` | Leaves the current voice channel |
| `speak_in_voice` | Speaks text aloud in the user's voice channel |

---

## 📊 Chart Types

Aira can generate charts inline in her response embed. Supported types:

| Type | Best for |
|---|---|
| `bar` | Comparisons and rankings |
| `horizontal_bar` | Long category names |
| `line` | Trends over time |
| `pie` | Proportions and percentages |
| `radar` | Multi-attribute comparisons |

Supports custom colors per dataset (hex), dark/light/grafana/ant themes, and custom width/height.

---

## 🧵 Built With

- [Go](https://golang.org/)
- [DiscordGo](https://github.com/bwmarrin/discordgo)
- [DeepSeek API](https://platform.deepseek.com/)
- [Exa Search](https://exa.ai/)
- [go-charts](https://github.com/vicanso/go-charts)
- [Piper TTS](https://github.com/rhasspy/piper)
- [hraban/opus](https://github.com/hraban/opus)

---

## 🤝 Contributing

Contributions, PRs, and sarcastic comments welcome.

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

---

## 🧠 Credits

Airo is inspired by **Aira from Dandadan** — chaotic, funny, and a little too smart for her own good.

<a href="https://github.com/jayydoesdev/airo/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=jayydoesdev/airo" />
</a>
