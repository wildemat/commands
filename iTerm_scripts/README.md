# iTerm project profiles

Each iTerm profile launches one project: tabs in set directories, each running setup commands.

## One-time setup
1. iTerm2 → Settings → General → Magic → enable **Python API**.
2. `python3 -m venv .venv && .venv/bin/pip install -r requirements.txt`
3. Symlink onto PATH: `sudo ln -s ~/Github/commands/iTerm_scripts/iterm-project /usr/local/bin/iterm-project`

## Add a project
1. Copy `projects/example.py` to `projects/<name>.py` and edit `TABS`.
2. iTerm → Settings → Profiles → new profile → General → **Send text at start**: `iterm-project <name>`

The tab opened by the profile becomes the first entry in `TABS`; the rest open as new tabs in the same window. The first tab's commands are typed in just after the launcher exits.
Running `.venv/bin/python launch.py <name>` outside iTerm's session env opens a new window instead.
