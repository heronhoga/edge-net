#!/usr/bin/env python3
"""Dependency-free Linux telemetry agent with opt-in, synchronous package updates."""

import hashlib
import json
import logging
import os
from pathlib import Path
import signal
import subprocess
import threading
import time
import urllib.error
import urllib.parse
import urllib.request

LOG = logging.getLogger("edgenet")
MAX_ARTIFACT = 64 * 1024 * 1024


class NoRedirect(urllib.request.HTTPRedirectHandler):
    # Never forward device credentials to a redirect destination.
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class Agent:
    def __init__(self):
        self.url = os.environ["EDGENET_URL"].rstrip("/")
        if urllib.parse.urlparse(self.url).scheme not in ("http", "https"):
            raise ValueError("EDGENET_URL must be an HTTP(S) URL")
        self.device_id = os.environ["EDGENET_DEVICE_ID"]
        self.token = os.environ["EDGENET_DEVICE_TOKEN"]
        self.target = os.environ.get("EDGENET_TARGET", "")
        self.installer = os.environ.get("EDGENET_INSTALLER", "")
        if self.installer and (not self.target or not os.path.isabs(self.installer)):
            raise ValueError("Updates require EDGENET_TARGET and an absolute EDGENET_INSTALLER path")
        self.directory = Path(os.environ.get("EDGENET_STATE_DIR", "~/.local/state/edgenet")).expanduser().resolve()
        self.directory.mkdir(parents=True, exist_ok=True, mode=0o700)
        # A second process must not race the installer or overwrite its journal.
        import fcntl
        self.lock_file = (self.directory / "agent.lock").open("w")
        fcntl.flock(self.lock_file, fcntl.LOCK_EX | fcntl.LOCK_NB)
        self.state_path = self.directory / "state.json"
        self.state = {"version": os.environ.get("EDGENET_VERSION", "unknown"), "receipt": None}
        if self.state_path.exists():
            self.state = json.loads(self.state_path.read_text())
        self.stop = threading.Event()
        self.previous_cpu = None
        self.opener = urllib.request.build_opener(NoRedirect())

    def save(self):
        temp = self.state_path.with_suffix(".tmp")
        with temp.open("w") as file:
            os.chmod(temp, 0o600)
            json.dump(self.state, file)
            file.flush()
            os.fsync(file.fileno())
        os.replace(temp, self.state_path)
        fd = os.open(self.directory, os.O_RDONLY)
        try:
            os.fsync(fd)
        finally:
            os.close(fd)

    def request(self, path, body=None):
        request = urllib.request.Request(
            self.url + path,
            data=json.dumps(body).encode() if body is not None else None,
            headers={"Authorization": "Bearer " + self.token,
                     "X-Device-ID": self.device_id, "Content-Type": "application/json"},
        )
        with self.opener.open(request, timeout=20) as response:
            raw = response.read(2 * 1024 * 1024)
            return json.loads(raw) if raw else None

    def metrics(self):
        metrics = {"uptime": int(float(Path("/proc/uptime").read_text().split()[0]))}
        # guest/guest_nice are already included in user/nice; don't double-count them.
        values = [int(x) for x in Path("/proc/stat").read_text().splitlines()[0].split()[1:9]]
        total, idle = sum(values), values[3] + values[4]
        if self.previous_cpu:
            elapsed = total - self.previous_cpu[0]
            if elapsed > 0:
                metrics["cpu"] = round(max(0, min(100, 100 * (1 - (idle - self.previous_cpu[1]) / elapsed))), 2)
        self.previous_cpu = (total, idle)
        memory = {line.split(":")[0]: int(line.split()[1]) for line in Path("/proc/meminfo").read_text().splitlines()}
        if memory.get("MemTotal") and "MemAvailable" in memory:
            metrics["memory"] = round(100 * (1 - memory["MemAvailable"] / memory["MemTotal"]), 2)
        temperature = Path("/sys/class/thermal/thermal_zone0/temp")
        if temperature.exists():
            try:
                metrics["temperature"] = float(temperature.read_text()) / 1000
            except (OSError, ValueError):
                pass
        return metrics

    def heartbeats(self):
        while not self.stop.is_set():
            try:
                self.request("/api/agent/heartbeat", {"version": self.state["version"], "metrics": self.metrics()})
                LOG.info("Heartbeat sent (%s)", self.state["version"])
            except Exception as error:
                LOG.warning("Heartbeat failed: %s", error)
            self.stop.wait(30)

    def report(self, job_id, status, message=""):
        self.request("/api/agent/jobs/" + job_id + "/status", {"status": status, "message": message[:500]})

    def finish(self, job_id, status, message):
        # Persist the outcome BEFORE sending it, so a lost response never repeats an install.
        self.state["receipt"] = {"id": job_id, "status": status, "message": message[:500]}
        self.save()
        self.flush_receipt()

    def flush_receipt(self):
        receipt = self.state.get("receipt")
        if not receipt:
            return
        if receipt["status"] == "installing":
            receipt.update(status="failed", message="Agent interrupted during installation; inspect device before retrying")
            self.save()
        self.report(receipt["id"], receipt["status"], receipt["message"])
        self.state["receipt"] = None
        self.save()

    def download(self, path, artifact):
        if not path.startswith("/api/agent/jobs/") or ".." in path:
            raise ValueError("Unexpected download path")
        destination = self.directory / "artifact.bin"
        temporary = self.directory / "artifact.part"
        digest, size = hashlib.sha256(), 0
        request = urllib.request.Request(self.url + path, headers={
            "Authorization": "Bearer " + self.token, "X-Device-ID": self.device_id,
        })
        try:
            with self.opener.open(request, timeout=60) as response, temporary.open("wb") as file:
                os.chmod(temporary, 0o600)
                while chunk := response.read(64 * 1024):
                    size += len(chunk)
                    if size > MAX_ARTIFACT:
                        raise ValueError("Artifact exceeds 64 MiB")
                    digest.update(chunk)
                    file.write(chunk)
                file.flush()
                os.fsync(file.fileno())
            if size != artifact["size"] or digest.hexdigest() != artifact["sha256"]:
                raise ValueError("Artifact size or SHA-256 checksum does not match")
            os.replace(temporary, destination)
            return destination
        finally:
            temporary.unlink(missing_ok=True)

    def process(self, item):
        job, artifact = item["deployment"], item["firmware"]
        job_id = job["id"]
        if artifact["target"] != self.target:
            self.finish(job_id, "failed", "Hardware target mismatch")
            return
        if job["status"] == "installing":
            self.finish(job_id, "failed", "Installation outcome is unknown; inspect device before retrying")
            return
        self.report(job_id, "downloading")
        try:
            path = self.download(item["downloadPath"], artifact)
        except (ValueError, OSError) as error:
            # Transport errors retry on the next poll; invalid artifacts fail permanently.
            if isinstance(error, (urllib.error.URLError, TimeoutError)):
                raise
            self.finish(job_id, "failed", str(error))
            return
        self.report(job_id, "installing")
        self.state["receipt"] = {"id": job_id, "status": "installing", "message": ""}
        self.save()
        LOG.info("Installing %s using %s", artifact["version"], self.installer)
        try:
            # No shell interpolation. The installer owns package extraction, activation,
            # health verification, and rollback. It must not reboot before returning.
            result = subprocess.run([self.installer, str(path), artifact["version"], artifact["target"]],
                                    timeout=600, check=False, cwd=self.directory)
            if result.returncode != 0:
                raise RuntimeError("Installer exited with code " + str(result.returncode))
        except (OSError, RuntimeError, subprocess.TimeoutExpired) as error:
            self.finish(job_id, "failed", str(error))
            return
        self.state["version"] = artifact["version"]
        self.finish(job_id, "succeeded", "Installer completed successfully")
        path.unlink(missing_ok=True)

    def run(self):
        worker = threading.Thread(target=self.heartbeats, daemon=True)
        worker.start()
        if not self.installer:
            LOG.info("Telemetry-only mode. Set EDGENET_INSTALLER and EDGENET_TARGET to enable updates.")
        while not self.stop.is_set():
            try:
                self.flush_receipt()
                if self.installer:
                    for item in self.request("/api/agent/jobs"):
                        if not self.stop.is_set():
                            self.process(item)
            except Exception as error:
                LOG.warning("Update poll failed (will retry): %s", error)
            self.stop.wait(30)
        worker.join(timeout=25)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    if not Path("/proc/stat").exists():
        raise SystemExit("This agent requires Linux (/proc).")
    agent = Agent()
    for sig in (signal.SIGTERM, signal.SIGINT):
        signal.signal(sig, lambda *_: agent.stop.set())
    agent.run()
