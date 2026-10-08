from lib import Tab

WINDOW_TITLE = "Kibana"

TABS = [
    Tab("app", "~/workplace/kibana", ["git status"]),
    Tab("chat", "~/workplace/kibana", ["claude --model claude-opus-5"]),
    Tab("kbn-dev", "~/workplace/kibana", ["which kbn-dev"]),
]
