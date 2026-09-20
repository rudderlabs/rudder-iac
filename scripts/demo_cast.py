#!/usr/bin/env python3
"""Record a command's terminal output to an asciicast v2 file over a PTY.

asciinema is not available in this environment (see Makefile's demo-cast
target), so this is a small stand-in written against the Python stdlib only.
It produces the same shape asciinema does for a v2 cast: a one-line JSON
header followed by JSONL `[elapsed_seconds, "o", data]` output events.

What this deliberately does NOT capture, unlike asciinema:

- Terminal-resize ("r") events. The pty's window size is fixed once, up
  front, and held constant for the whole recording.
- Input ("i") events. Nothing is ever written to the child's stdin.

Both are fine to omit here because the only thing this script ever records
is demos/<Test>/demo.sh: a generated, non-interactive script that writes
output and reads no input, run once from start to finish with no resize.
A general-purpose recorder would need both; this one doesn't, so it skips
the complexity rather than stub it out unused.
"""
from __future__ import annotations

import argparse
import fcntl
import json
import os
import pty
import select
import struct
import sys
import termios
import time


def record(argv: list[str], cast_path: str, width: int, height: int,
           header_extra: dict | None = None) -> int:
    """Run argv under a PTY, writing an asciicast v2 file to cast_path.

    Returns the child's exit code.
    """
    pid, master_fd = pty.fork()
    if pid == 0:
        # Child: pty.fork() already made us the session leader with the
        # slave as our controlling tty, so a plain exec is all that's left.
        try:
            os.execvp(argv[0], argv)
        except OSError as exc:
            os.write(2, f"demo_cast: exec {argv[0]}: {exc}\n".encode())
            os._exit(127)

    # Fix the pty's reported size so the recording doesn't depend on
    # whatever terminal happens to be running this script.
    fcntl.ioctl(master_fd, termios.TIOCSWINSZ,
                struct.pack("HHHH", height, width, 0, 0))

    events: list[list] = []
    start = time.monotonic()
    exit_status = 0
    try:
        while True:
            try:
                readable, _, _ = select.select([master_fd], [], [], 1.0)
            except InterruptedError:
                continue

            if master_fd in readable:
                try:
                    data = os.read(master_fd, 65536)
                except OSError:
                    # EIO here means the slave side closed (child exited).
                    data = b""
                if data:
                    events.append([round(time.monotonic() - start, 6), "o",
                                    data.decode("utf-8", "replace")])
                else:
                    break

            wpid, status = os.waitpid(pid, os.WNOHANG)
            if wpid == pid:
                # Drain whatever landed between the last read and exit.
                while True:
                    try:
                        data = os.read(master_fd, 65536)
                    except OSError:
                        break
                    if not data:
                        break
                    events.append([round(time.monotonic() - start, 6), "o",
                                    data.decode("utf-8", "replace")])
                exit_status = os.waitstatus_to_exitcode(status)
                break
    finally:
        os.close(master_fd)

    header = {
        "version": 2,
        "width": width,
        "height": height,
        "env": {
            "TERM": os.environ.get("TERM", "xterm-256color"),
            "SHELL": os.environ.get("SHELL", "/bin/bash"),
        },
    }
    if header_extra:
        header.update(header_extra)

    with open(cast_path, "w") as f:
        f.write(json.dumps(header) + "\n")
        for event in events:
            f.write(json.dumps(event) + "\n")

    return exit_status


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("script", help="executable to run under the PTY")
    ap.add_argument("cast_out", help="path to write the .cast file to")
    ap.add_argument("--manifest",
                     help="demo manifest.json to embed as header.rudderDemo")
    ap.add_argument("--width", type=int, default=118)
    ap.add_argument("--height", type=int, default=34)
    args = ap.parse_args()

    header_extra = {}
    if args.manifest:
        with open(args.manifest) as f:
            header_extra["rudderDemo"] = json.load(f)

    exit_code = record([args.script], args.cast_out, args.width, args.height,
                        header_extra=header_extra)
    if exit_code != 0:
        print(f"demo_cast: {args.script} exited {exit_code}", file=sys.stderr)
    return 0 if exit_code == 0 else exit_code


if __name__ == "__main__":
    sys.exit(main())
