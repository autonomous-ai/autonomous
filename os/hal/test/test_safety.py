"""Tests for the safety policy layer — pure parser + gate, no hardware.

Covers the slice-1 (brightness ceiling) checklist in docs/safety.md:
unit clamp behavior, schema fail-loud, and the load_safety fail-safe rules.
"""
import os
import tempfile
import unittest

from hal.safety.policy import (
    SafetyPolicy,
    clamp_brightness,
    clamp_color,
    load_safety,
    parse_safety,
    validate_schema,
)

_FM = "---\nschema: autonomous.safety.v1\nlight:\n  max_brightness: 180\n---\n# prose\n"


class TestParse(unittest.TestCase):
    def test_parse_full(self):
        p = parse_safety(_FM)
        self.assertEqual(p.schema, "autonomous.safety.v1")
        self.assertEqual(p.max_brightness, 180)

    def test_parse_no_light_bound(self):
        p = parse_safety("---\nschema: autonomous.safety.v1\n---\n")
        self.assertEqual(p.max_brightness, None)

    def test_parse_flow_style(self):
        p = parse_safety("---\nschema: autonomous.safety.v1\nlight: { max_brightness: 90 }\n---\n")
        self.assertEqual(p.max_brightness, 90)

    def test_out_of_range_raises(self):
        with self.assertRaises(ValueError):
            parse_safety("---\nschema: autonomous.safety.v1\nlight:\n  max_brightness: 300\n---\n")


class TestSchemaValidation(unittest.TestCase):
    def test_missing_schema_raises(self):
        with self.assertRaises(ValueError):
            validate_schema("light:\n  max_brightness: 180\n")

    def test_malformed_schema_raises(self):
        with self.assertRaises(ValueError):
            validate_schema("schema: not.a.valid.tag\n")

    def test_unknown_major_raises(self):
        with self.assertRaises(ValueError):
            validate_schema("schema: autonomous.safety.v2\n")

    def test_valid_schema_passes(self):
        self.assertEqual(validate_schema("schema: autonomous.safety.v1\n"), "autonomous.safety.v1")


class TestClampBrightness(unittest.TestCase):
    def setUp(self):
        self.p = SafetyPolicy(schema="autonomous.safety.v1", max_brightness=180)

    def test_above_ceiling_clamps(self):
        self.assertEqual(clamp_brightness(self.p, 255), 180)

    def test_below_ceiling_passes(self):
        self.assertEqual(clamp_brightness(self.p, 120), 120)

    def test_no_policy_passes_through(self):
        self.assertEqual(clamp_brightness(None, 255), 255)

    def test_no_ceiling_passes_through(self):
        p = SafetyPolicy(schema="autonomous.safety.v1", max_brightness=None)
        self.assertEqual(clamp_brightness(p, 255), 255)


class TestClampColor(unittest.TestCase):
    def setUp(self):
        self.p = SafetyPolicy(schema="autonomous.safety.v1", max_brightness=180)

    def test_full_white_clamps_to_ceiling(self):
        self.assertEqual(clamp_color(self.p, (255, 255, 255)), (180, 180, 180))

    def test_hue_preserved_when_scaling(self):
        # pure red at full -> scaled to ceiling, still pure red
        self.assertEqual(clamp_color(self.p, (255, 0, 0)), (180, 0, 0))

    def test_below_ceiling_unchanged(self):
        self.assertEqual(clamp_color(self.p, (100, 50, 0)), (100, 50, 0))

    def test_no_policy_passes_through(self):
        self.assertEqual(clamp_color(None, (255, 255, 255)), (255, 255, 255))


class TestLoadSafety(unittest.TestCase):
    def _write(self, name, text):
        d = tempfile.mkdtemp()
        with open(os.path.join(d, name), "w") as f:
            f.write(text)
        return d

    def test_no_safety_ref_returns_none(self):
        self.assertIsNone(load_safety("/tmp", ""))

    def test_valid_file_loads(self):
        d = self._write("SAFETY.md", _FM)
        p = load_safety(d, "SAFETY.md")
        self.assertEqual(p.max_brightness, 180)

    def test_prose_only_returns_none(self):
        # SAFETY.md with no front matter -> pass-through (legacy prose), not a crash
        d = self._write("SAFETY.md", "# SAFETY.md\n\nNo front matter here.\n")
        self.assertIsNone(load_safety(d, "SAFETY.md"))

    def test_missing_file_returns_none(self):
        # declared safety_ref but no file -> pass-through + warn, not a crash
        self.assertIsNone(load_safety(tempfile.mkdtemp(), "SAFETY.md"))

    def test_bad_schema_file_raises(self):
        # present front matter with an unknown major -> fail loud (abort boot)
        d = self._write("SAFETY.md", "---\nschema: autonomous.safety.v9\n---\n")
        with self.assertRaises(ValueError):
            load_safety(d, "SAFETY.md")


if __name__ == "__main__":
    unittest.main()
