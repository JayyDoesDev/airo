<p align="center"><img src="https://github.com/JayyDoesDev/airo/blob/main/.github/assets/aira.png" alt="aira" width="400"/></p>
<h1 align="center">Aira</h1>
<h2 align="center">💬✨ Your sassy AI sidekick — powered by DiscordGo + Claude or GPT</h2>

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

- 🧠 **AI Chat** – Natural and cheeky replies like Aira from *Dandadan*
- 🚀 **Tasks** – Auto-performs Discord actions (kick, ban, DM, assign role, etc.)
- 🖼️ **Embeds** – Sends beautiful responses with titles, thumbnails, and images
- 🔀 **Queue System** – Executes all tasks asynchronously and in order
- 🤖 **Zero Slash Needed** – Just mention the bot and it does the rest
- 📝 **Memory** - Can remember things and will create memories it thinks it will need. (binary based)

---

## 🖼 Preview

> Replace these image links with real bot screenshots in `.github/assets`

### AI + Task Response  
<p align="center"><img src="https://github.com/jayydoesdev/airo/blob/main/.github/assets/ai-response.png?raw=true" alt="AI Response Preview" width="700"/></p>

### Embed Role List  
<p align="center"><img src="https://github.com/jayydoesdev/airo/blob/main/.github/assets/role-list-embed.png?raw=true" alt="Role List Embed" width="700"/></p>

---

## 📁 Project Structure

```
airo/
├── bot/
│   ├── discord/       # DiscordGo 
│   ├── lib/          # AI prompts, JSON parsing, utilities
│   ├── tasks/        # Task queue + execution logic
│   └── main.go       # Bot entry point
├── .env              # Local environment vars (OPENAPI key, Discord token)
└── go.mod            # Go module definition
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
ANTHROPIC_API_KEY=your_anthroptic_api_key
OPENAI_API_KEY=your_openai_api_key
DISCORD_BOT_TOKEN=your_bot_token
GUILD_ID=your_guild_id
```

> Ensure you enable **MESSAGE CONTENT INTENT** and **GUILD MEMBERS INTENT** in the Discord Developer Portal.

### 3. Run the bot

```bash
go run main.go
```

---

## ⚙️ Example Prompt

Your bot speaks with flair and includes hidden JSON instructions like this:

```plaintext
Whoa whoa—someone's getting kicked. 🍿

{
  "response_type": "message",
  "response": "Request handled. Try not to cause chaos again.",
  "tasks": [
    {
      "action": "kick_user",
      "target_user": "123456789012345678",
      "reason": "spamming"
    },
    {
      "action": "dm_user",
      "target_user": "123456789012345678",
      "dm_content": "You've been kicked from the server. Maybe calm down a bit next time?"
    }
  ]
}
```

---

## 🧪 Commands & Behaviors

| Action           | Description                          |
|------------------|--------------------------------------|
| `kick_user`      | Kicks a member from the server       |
| `ban_user`       | Bans a member                        |
| `assign_role`    | Gives a user a role                  |
| `remove_role`    | Removes a role from a user           |
| `dm_user`        | Sends a private message              |
| `list_user_roles`| Lists roles for a specific member    |

---

## 🧵 Built With

- [Go](https://golang.org/)
- [DiscordGo](https://github.com/bwmarrin/discordgo)
- [Claude / OpenAI API](https://platform.openai.com/docs)

---

## 🧩 Future Plans

- [ ] Slash command integration
- [ ] Message logging
- [ ] Custom prompts per server
- [ ] Local file-based config + admin interface

---

## 🤝 Contributing

Contributions, PRs, and sarcastic comments welcome.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## 📄 License

This project is licensed under the MIT License — see [LICENSE](LICENSE) for details.

---

## 🧠 Credits

Airo is inspired by **Aira from Dandadan** — chaotic, funny, and a little too smart for her own good.

<a href="https://github.com/jayydoesdev/airo/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=jayydoesdev/airo" />
</a>
