#!/usr/bin/env python3
"""End-to-end probe for NOVA managed commercial ingress listeners.

The probe reads the already-generated Xray runtime config, builds an isolated
loopback client for each supported managed listener, and makes an HTTPS request
through the complete managed-listener -> commercial-outbound path. Credentials
remain in memory or a mode-0600 temporary directory and are never printed.
"""

from __future__ import annotations

import json
import os
import pathlib
import socket
import subprocess
import sys
import tempfile
import time
from typing import Any


CONFIG_PATH = pathlib.Path("/usr/local/x-ui/bin/config.json")
TEST_URL = "https://www.gstatic.com/generate_204"
SUPPORTED = {"vless", "trojan", "hysteria"}


def fail(message: str) -> None:
    print(f"ERROR: {message}", file=sys.stderr)
    raise SystemExit(1)


def find_xray() -> pathlib.Path:
    candidates = sorted(CONFIG_PATH.parent.glob("xray-linux-*"))
    for candidate in candidates:
        if candidate.is_file() and os.access(candidate, os.X_OK):
            return candidate
    fail("packaged Xray binary was not found")


def free_tcp_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
        listener.bind(("127.0.0.1", 0))
        return int(listener.getsockname()[1])


def wait_tcp_port(port: int, process: subprocess.Popen[str]) -> None:
    deadline = time.monotonic() + 6
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError("temporary Xray client exited before opening its proxy")
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as client:
            client.settimeout(0.3)
            if client.connect_ex(("127.0.0.1", port)) == 0:
                return
        time.sleep(0.15)
    raise RuntimeError("temporary Xray client did not open its local proxy")


def reality_public_key(xray: pathlib.Path, private_key: str) -> str:
    result = subprocess.run(
        [str(xray), "x25519", "-i", private_key],
        check=True,
        capture_output=True,
        text=True,
        timeout=10,
    )
    values = [
        line.split(":", 1)[1].strip()
        for line in result.stdout.splitlines()
        if ":" in line and line.split(":", 1)[1].strip()
    ]
    if len(values) < 2:
        raise RuntimeError("Xray did not derive the Reality public key")
    return values[1]


def first_client(inbound: dict[str, Any]) -> dict[str, Any]:
    clients = inbound.get("settings", {}).get("clients", [])
    if not isinstance(clients, list) or not clients or not isinstance(clients[0], dict):
        raise RuntimeError("managed listener has no runtime client")
    return clients[0]


def tls_client_settings(inbound: dict[str, Any]) -> dict[str, Any]:
    server = inbound.get("streamSettings", {}).get("tlsSettings", {})
    server_name = str(server.get("serverName", "")).strip()
    if not server_name:
        raise RuntimeError("managed TLS listener has no serverName")
    result: dict[str, Any] = {
        "serverName": server_name,
        "verifyPeerCertByName": server_name,
        "fingerprint": "chrome",
        "allowInsecure": False,
    }
    if isinstance(server.get("alpn"), list):
        result["alpn"] = server["alpn"]
    return result


def client_outbound(inbound: dict[str, Any], xray: pathlib.Path) -> dict[str, Any]:
    protocol = str(inbound.get("protocol", "")).lower()
    port = int(inbound["port"])
    client = first_client(inbound)
    stream = inbound.get("streamSettings", {})
    managed_websocket = (
        str(stream.get("network", "")).lower() == "ws"
        and str(stream.get("security", "")).lower() == "none"
    )
    websocket_settings = stream.get("wsSettings", {})
    websocket_path = str(websocket_settings.get("path", "")).strip()
    websocket_host = str(websocket_settings.get("host", "")).strip()
    if managed_websocket and not websocket_path:
        raise RuntimeError("managed WebSocket listener has no path")

    if protocol == "trojan":
        password = str(client.get("password", ""))
        if not password:
            raise RuntimeError("Trojan runtime client has no password")
        if managed_websocket:
            return {
                "protocol": "trojan",
                "tag": "managed-probe-out",
                "settings": {
                    "servers": [
                        {"address": "127.0.0.1", "port": port, "password": password}
                    ]
                },
                "streamSettings": {
                    "network": "ws",
                    "security": "none",
                    "wsSettings": {
                        "path": websocket_path,
                        "host": websocket_host,
                    },
                },
            }
        return {
            "protocol": "trojan",
            "tag": "managed-probe-out",
            "settings": {
                "servers": [
                    {"address": "127.0.0.1", "port": port, "password": password}
                ]
            },
            "streamSettings": {
                "network": "tcp",
                "security": "tls",
                "tcpSettings": {"header": {"type": "none"}},
                "tlsSettings": tls_client_settings(inbound),
            },
        }

    if protocol == "vless":
        client_id = str(client.get("id", ""))
        if not client_id:
            raise RuntimeError("VLESS runtime client has no id")
        if managed_websocket:
            return {
                "protocol": "vless",
                "tag": "managed-probe-out",
                "settings": {
                    "vnext": [
                        {
                            "address": "127.0.0.1",
                            "port": port,
                            "users": [{"id": client_id, "encryption": "none"}],
                        }
                    ]
                },
                "streamSettings": {
                    "network": "ws",
                    "security": "none",
                    "wsSettings": {
                        "path": websocket_path,
                        "host": websocket_host,
                    },
                },
            }
        reality = stream.get("realitySettings", {})
        private_key = str(reality.get("privateKey", ""))
        server_names = reality.get("serverNames", [])
        short_ids = reality.get("shortIds", [])
        if not private_key or not server_names or not short_ids:
            raise RuntimeError("VLESS Reality listener settings are incomplete")
        user: dict[str, Any] = {"id": client_id, "encryption": "none"}
        if str(client.get("flow", "")):
            user["flow"] = client["flow"]
        return {
            "protocol": "vless",
            "tag": "managed-probe-out",
            "settings": {
                "vnext": [
                    {"address": "127.0.0.1", "port": port, "users": [user]}
                ]
            },
            "streamSettings": {
                "network": "tcp",
                "security": "reality",
                "tcpSettings": {"header": {"type": "none"}},
                "realitySettings": {
                    "show": False,
                    "fingerprint": "chrome",
                    "serverName": str(server_names[0]),
                    "publicKey": reality_public_key(xray, private_key),
                    "shortId": str(short_ids[0]),
                    "spiderX": "/",
                },
            },
        }

    if protocol == "hysteria":
        auth = str(client.get("auth", ""))
        if not auth:
            raise RuntimeError("Hysteria2 runtime client has no auth")
        return {
            "protocol": "hysteria",
            "tag": "managed-probe-out",
            "settings": {"address": "127.0.0.1", "port": port, "version": 2},
            "streamSettings": {
                "network": "hysteria",
                "security": "tls",
                "tlsSettings": tls_client_settings(inbound),
                "hysteriaSettings": {
                    "version": 2,
                    "auth": auth,
                    "udpIdleTimeout": 60,
                },
            },
        }

    raise RuntimeError(f"unsupported probe protocol {protocol!r}")


def sanitized_log(log_path: pathlib.Path) -> str:
    if not log_path.exists():
        return "(no temporary Xray log)"
    lines = log_path.read_text(encoding="utf-8", errors="replace").splitlines()
    return "\n".join(lines[-20:]) or "(empty temporary Xray log)"


def probe(
    inbound: dict[str, Any], xray: pathlib.Path, workdir: pathlib.Path
) -> tuple[bool, str]:
    protocol = str(inbound.get("protocol", "")).lower()
    public_port = int(inbound["port"])
    proxy_port = free_tcp_port()
    config = {
        "log": {"loglevel": "warning"},
        "inbounds": [
            {
                "listen": "127.0.0.1",
                "port": proxy_port,
                "protocol": "http",
                "tag": "managed-probe-http",
                "settings": {},
            }
        ],
        "outbounds": [client_outbound(inbound, xray)],
    }
    config_path = workdir / f"{protocol}-{public_port}.json"
    log_path = workdir / f"{protocol}-{public_port}.log"
    config_path.write_text(json.dumps(config), encoding="utf-8")
    config_path.chmod(0o600)

    with log_path.open("w", encoding="utf-8") as log_file:
        process = subprocess.Popen(
            [str(xray), "-c", str(config_path)],
            stdout=log_file,
            stderr=subprocess.STDOUT,
            text=True,
        )
        try:
            wait_tcp_port(proxy_port, process)
            result = subprocess.run(
                [
                    "curl",
                    "-sS",
                    "-o",
                    "/dev/null",
                    "-w",
                    "%{http_code}",
                    "--connect-timeout",
                    "8",
                    "--max-time",
                    "25",
                    "--proxy",
                    f"http://127.0.0.1:{proxy_port}",
                    TEST_URL,
                ],
                capture_output=True,
                text=True,
                timeout=30,
            )
            status = result.stdout.strip()
            if result.returncode == 0 and status.startswith(("2", "3")):
                return True, f"HTTP {status}"
            detail = result.stderr.strip() or f"curl exit {result.returncode}, HTTP {status}"
            return False, f"{detail}\n{sanitized_log(log_path)}"
        except Exception as exc:  # diagnostics must report every protocol
            return False, f"{exc}\n{sanitized_log(log_path)}"
        finally:
            process.terminate()
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=3)


def main() -> int:
    if os.geteuid() != 0:
        fail("run this diagnostic as root")
    if not CONFIG_PATH.exists():
        fail(f"{CONFIG_PATH} does not exist")
    xray = find_xray()
    config = json.loads(CONFIG_PATH.read_text(encoding="utf-8"))
    inbounds = [
        inbound
        for inbound in config.get("inbounds", [])
        if str(inbound.get("tag", "")).startswith("commercial-in-")
        and str(inbound.get("protocol", "")).lower() in SUPPORTED
    ]
    if not inbounds:
        fail("no supported managed commercial listeners were found")

    failures = 0
    with tempfile.TemporaryDirectory(prefix="nova-managed-probe-") as directory:
        workdir = pathlib.Path(directory)
        for inbound in sorted(inbounds, key=lambda item: int(item.get("port", 0))):
            protocol = str(inbound.get("protocol", "")).lower()
            port = int(inbound.get("port", 0))
            try:
                passed, detail = probe(inbound, xray, workdir)
            except Exception as exc:
                passed, detail = False, str(exc)
            label = "PASS" if passed else "FAIL"
            print(f"{label} {protocol} port={port}: {detail}")
            if not passed:
                failures += 1
    print(f"SUMMARY: passed={len(inbounds) - failures} failed={failures}")
    return 0 if failures == 0 else 2


if __name__ == "__main__":
    raise SystemExit(main())
