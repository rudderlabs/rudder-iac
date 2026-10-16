#!/usr/bin/env python3
"""Tests for demo_cast.py's PTY recorder.

Structural checks (JSON parses, timestamps monotonic, event shape) can't
catch either bug these tests target: a decoding bug and a missing-env-var
bug are both invisible to a parser that only looks at the container, not
the rendered text. So these reconstruct the cast's visible text — the way
a human actually reads it — and assert against that.

Run directly: python3 scripts/demo_cast_test.py
"""
from __future__ import annotations

import importlib.util
import json
import os
import stat
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
_spec = importlib.util.spec_from_file_location(
    "demo_cast", os.path.join(HERE, "demo_cast.py"))
demo_cast = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(demo_cast)


def record_script(body: str, **record_kwargs):
    """Write `body` as a shell script, record it, and return (header, text).

    `text` is every event's data concatenated in order — the cast's visible
    content, independent of how it happened to be chunked into events.
    """
    with tempfile.TemporaryDirectory() as d:
        script_path = os.path.join(d, "run.sh")
        with open(script_path, "w") as f:
            f.write("#!/bin/sh\n" + body)
        os.chmod(script_path, os.stat(script_path).st_mode | stat.S_IEXEC)

        cast_path = os.path.join(d, "out.cast")
        demo_cast.record([script_path], cast_path, width=80, height=24,
                          **record_kwargs)

        with open(cast_path) as f:
            lines = f.read().splitlines()
        header = json.loads(lines[0])
        events = [json.loads(line) for line in lines[1:]]
        text = "".join(ev[2] for ev in events)
        return header, text


class MultibyteTests(unittest.TestCase):
    # em dash, ellipsis, check mark, arrow — the exact glyphs the demo
    # generator's "verified in Go" footer and demo-magic's prompt use.
    GLYPHS = "— … ✔ ➜"

    def test_multibyte_survives_default_chunking(self):
        header, text = record_script(f"printf '%s' '{self.GLYPHS}'\n")
        self.assertNotIn("�", text, "U+FFFD (replacement char) in cast")
        for glyph in self.GLYPHS.split(" "):
            self.assertIn(glyph, text)

    def test_multibyte_survives_forced_read_boundary(self):
        # read_size=1 guarantees every multi-byte UTF-8 sequence (2-3 bytes
        # here) is split across multiple os.read() calls — this doesn't
        # pass because a boundary happened not to land mid-rune, it passes
        # despite one landing mid-rune on every single character.
        header, text = record_script(f"printf '%s' '{self.GLYPHS}'\n",
                                       read_size=1)
        self.assertNotIn("�", text, "U+FFFD (replacement char) in cast")
        for glyph in self.GLYPHS.split(" "):
            self.assertIn(glyph, text)


class TermEnvTests(unittest.TestCase):
    def test_term_is_set_and_reported_truthfully(self):
        header, text = record_script("echo TERM=$TERM\n")
        self.assertNotIn("TERM environment variable not set", text)
        self.assertIn(f"TERM={header['env']['TERM']}", text)
        self.assertTrue(header["env"]["TERM"], "header TERM must be non-empty")

    def test_existing_term_is_preserved_not_overwritten(self):
        old = os.environ.get("TERM")
        try:
            os.environ["TERM"] = "screen-256color"
            header, text = record_script("echo TERM=$TERM\n")
        finally:
            if old is None:
                os.environ.pop("TERM", None)
            else:
                os.environ["TERM"] = old
        self.assertEqual(header["env"]["TERM"], "screen-256color")
        self.assertIn("TERM=screen-256color", text)


if __name__ == "__main__":
    unittest.main()
