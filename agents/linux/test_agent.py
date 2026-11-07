"""Tests for update integrity and crash recovery; no actual hardware is modified."""

import hashlib
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from agent import Agent


class AgentTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        env = patch.dict(os.environ, {
            "EDGENET_URL": "http://localhost:8080",
            "EDGENET_DEVICE_ID": "test-device",
            "EDGENET_DEVICE_TOKEN": "test-token",
            "EDGENET_STATE_DIR": self.temp.name,
            "EDGENET_TARGET": "linux-arm64",
            "EDGENET_INSTALLER": "/test/installer",
        })
        env.start()
        self.addCleanup(env.stop)
        self.agent = Agent()
        self.addCleanup(self.agent.lock_file.close)

    def item(self, status="queued"):
        return {"deployment": {"id": "job-1", "status": status},
                "firmware": {"target": "linux-arm64", "version": "2.0.0", "size": 7,
                             "sha256": hashlib.sha256(b"package").hexdigest()},
                "downloadPath": "/api/agent/jobs/job-1/artifact"}

    def test_checksum_failure_never_runs_installer(self):
        with patch.object(self.agent, "report") as report, \
                patch.object(self.agent.opener, "open", return_value=io.BytesIO(b"corrupt")), \
                patch("agent.subprocess.run") as installer:
            self.agent.process(self.item())
            installer.assert_not_called()
            self.assertEqual(report.call_args.args[1], "failed")
            self.assertIn("checksum", report.call_args.args[2])
            self.assertFalse((Path(self.temp.name) / "artifact.bin").exists())

    def test_success_persists_receipt_before_ack_and_does_not_reinstall(self):
        def report(job, status, message=""):
            if status == "succeeded":
                saved = json.loads(self.agent.state_path.read_text())
                self.assertEqual(saved["receipt"]["status"], "succeeded")
                raise ConnectionError("lost response")

        with patch.object(self.agent, "report", side_effect=report), \
                patch.object(self.agent.opener, "open", return_value=io.BytesIO(b"package")), \
                patch("agent.subprocess.run") as installer:
            installer.return_value.returncode = 0
            with self.assertRaises(ConnectionError):
                self.agent.process(self.item())
            installer.assert_called_once()
            self.assertEqual(self.agent.state["version"], "2.0.0")
            # After restart/network recovery, only acknowledge the persisted receipt.
            with patch.object(self.agent, "report") as retry:
                self.agent.flush_receipt()
                retry.assert_called_once_with("job-1", "succeeded", "Installer completed successfully")
            installer.assert_called_once()
            self.assertIsNone(self.agent.state["receipt"])

    def test_interrupted_install_is_failed_not_retried(self):
        self.agent.state["receipt"] = {"id": "job-1", "status": "installing", "message": ""}
        self.agent.save()
        with patch.object(self.agent, "report") as report, patch("agent.subprocess.run") as installer:
            self.agent.flush_receipt()
            self.assertEqual(report.call_args.args[1], "failed")
            self.assertIn("interrupted", report.call_args.args[2])
            installer.assert_not_called()

    def test_wrong_target_never_downloads(self):
        item = self.item()
        item["firmware"]["target"] = "esp32"
        with patch.object(self.agent, "report") as report, patch.object(self.agent, "download") as download:
            self.agent.process(item)
            download.assert_not_called()
            report.assert_called_once_with("job-1", "failed", "Hardware target mismatch")


if __name__ == "__main__":
    unittest.main()
