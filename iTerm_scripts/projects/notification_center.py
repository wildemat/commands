from lib import Tab

WINDOW_TITLE = "Notification Center"

TABS = [
    Tab("app", "~/workplace/kibana", ["git status"]),
    Tab("chat", "~/workplace/kibana", ["claude --model claude-opus-5"]),
    Tab("kbn-dev", "~/workplace/kibana", ["which kbn-dev"]),
    Tab("chat-temp", "~/workplace/kibana", ["claude '@/Users/wildmat/Github/obsidian-notes/the_collection/Work/NotificationCenterWork'"]),
]
