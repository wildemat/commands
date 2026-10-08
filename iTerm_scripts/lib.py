"""Shared machinery for iTerm project launchers.

A project file in projects/ defines TABS. The tab that owns the invoking
session is configured by emitting shell lines for the caller to eval; the rest
are opened through the iTerm2 Python API.
"""
import os
import shlex
import subprocess
import sys
from dataclasses import dataclass, field

import iterm2


@dataclass
class Tab:
    title: str
    cwd: str
    commands: list[str] = field(default_factory=list)


def _setup_text(tab: Tab) -> str:
    lines = [f"cd {shlex.quote(os.path.expanduser(tab.cwd))}", *tab.commands]
    return "\n".join(lines) + "\n"


async def _open_project(
    connection, tabs: list[Tab], current_session_id: str | None, window_title: str | None
) -> bool:
    """Returns True if the invoking session was reused for tabs[0]."""
    app = await iterm2.async_get_app(connection)
    current = app.get_session_by_id(current_session_id) if current_session_id else None
    window = None
    if current:
        window = next(
            (w for w in app.terminal_windows if any(t.tab_id == current.tab.tab_id for t in w.tabs)),
            None,
        )
    # A profile opened as a tab lands in an existing window; the project gets
    # its own window and the stray tab is closed. A lone tab is already a
    # fresh window and is reused.
    stray = current if window is not None and len(window.tabs) > 1 else None
    first_in_new_window = window is None or stray is not None
    if first_in_new_window:
        window = await iterm2.Window.async_create(connection)
        current = window.current_tab.current_session

    if window_title:
        await window.async_set_title(window_title)
    await current.tab.async_set_title(tabs[0].title)
    if first_in_new_window:
        await current.async_send_text(_setup_text(tabs[0]))
    remaining = tabs[1:]

    for tab in remaining:
        new = await window.async_create_tab()
        await new.async_set_title(tab.title)
        await new.current_session.async_send_text(_setup_text(tab))

    await current.tab.async_select()
    if stray:
        await stray.async_close(force=True)
    return not first_in_new_window


def open_project(tabs: list[Tab], window_title: str | None = None) -> None:
    """Open `tabs`, reusing the invoking session for the first one.

    """
    session_id = os.environ.get("ITERM_SESSION_ID", "").split(":")[-1] or None

    reused = False

    async def main(connection):
        nonlocal reused
        reused = await _open_project(connection, tabs, session_id, window_title)

    iterm2.run_until_complete(main)
    if reused:
        _send_after_exit(session_id, _setup_text(tabs[0]))


def _send_after_exit(session_id: str, text: str) -> None:
    """Type `text` into the invoking session once this process has exited.

    Sending while we still own the foreground would echo the text twice, and a
    child process can't alter the parent shell's cwd any other way.
    """
    code = (
        "import sys, time, iterm2\n"
        "time.sleep(0.4)\n"
        "async def main(c):\n"
        "    app = await iterm2.async_get_app(c)\n"
        "    await app.get_session_by_id(sys.argv[1]).async_send_text(sys.argv[2])\n"
        "iterm2.run_until_complete(main)\n"
    )
    subprocess.Popen(
        [sys.executable, "-c", code, session_id, text],
        start_new_session=True,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
