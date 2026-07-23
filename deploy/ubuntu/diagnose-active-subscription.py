#!/usr/bin/env python3
"""Validate active subscription links against the live Xray runtime.

The script fetches active raw subscriptions through the loopback-only
subscription service, validates their public parameters and credentials against
the generated Xray runtime, then performs an HTTPS request through each exact
subscription link. It never prints subscription IDs or credentials.
"""

from __future__ import annotations

import base64
import json
import os
import pathlib
import socket
import subprocess
import sys
import tempfile
import time
import urllib.parse
import urllib.request
from typing import Any


CONFIG_PATH = pathlib.Path("/usr/local/x-ui/bin/config.json")
DEPLOY_ENV_PATH = pathlib.Path("/etc/nova/deploy.env")
TEST_URL = "https://www.gstatic.com/generate_204"
SCHEME_PROTOCOL = {
    "vless": "vless",
    "trojan": "trojan",
    "hysteria2": "hysteria",
    "hy2": "hysteria",
}


def fail(message: str) -> None:
    print(f"ERROR: {message}", file=sys.stderr)
    raise SystemExit(1)


def load_deploy_env() -> dict[str, str]:
    if not DEPLOY_ENV_PATH.exists():
        fail(f"{DEPLOY_ENV_PATH} does not exist")
    result: dict[str, str] = {}
    for raw in DEPLOY_ENV_PATH.read_text(encoding="utf-8").splitlines():
        raw = raw.strip()
        if not raw or raw.startswith("#") or "=" not in raw:
            continue
        key, value = raw.split("=", 1)
        result[key.strip()] = value.strip().strip("\"'")
    return result


def required(env: dict[str, str], key: str) -> str:
    value = env.get(key, "").strip()
    if not value:
        fail(f"{key} is missing from {DEPLOY_ENV_PATH}")
    return value


def find_xray() -> pathlib.Path:
    for candidate in sorted(CONFIG_PATH.parent.glob("xray-linux-*")):
        if candidate.is_file() and os.access(candidate, os.X_OK):
            return candidate
    fail("packaged Xray binary was not found")


def active_subscription_ids(env: dict[str, str]) -> list[str]:
    sql = """
SELECT subscription_id
FROM commercial_subscription_entitlements
WHERE status = 'active'
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY updated_at DESC
LIMIT 10
""".strip()
    process_env = os.environ.copy()
    process_env["PGPASSWORD"] = required(env, "NOVA_DB_PASSWORD")
    result = subprocess.run(
        [
            "psql",
            "-h",
            "127.0.0.1",
            "-U",
            required(env, "NOVA_DB_USER"),
            "-d",
            required(env, "NOVA_DB_NAME"),
            "-AtX",
            "-v",
            "ON_ERROR_STOP=1",
            "-c",
            sql,
        ],
        check=True,
        capture_output=True,
        text=True,
        timeout=15,
        env=process_env,
    )
    return [line.strip() for line in result.stdout.splitlines() if line.strip()]


def subscription_url(env: dict[str, str], subscription_id: str) -> str:
    path = required(env, "NOVA_SUB_PATH")
    if not path.startswith("/"):
        path = "/" + path
    if not path.endswith("/"):
        path += "/"
    port = int(required(env, "NOVA_SUB_PORT"))
    return (
        f"http://127.0.0.1:{port}{path}"
        + urllib.parse.quote(subscription_id, safe="")
    )


def decode_subscription(body: bytes) -> list[str]:
    raw = body.strip()
    candidates = [raw]
    try:
        compact = b"".join(raw.split())
        compact += b"=" * ((4 - len(compact) % 4) % 4)
        candidates.append(base64.b64decode(compact, validate=False))
    except Exception:
        pass
    for candidate in candidates:
        text = candidate.decode("utf-8", errors="replace")
        links = [
            line.strip()
            for line in text.splitlines()
            if "://" in line and not line.lstrip().startswith("#")
        ]
        if links:
            return links
    raise RuntimeError("raw subscription did not contain recognizable links")


def fetch_subscription(env: dict[str, str], subscription_id: str) -> list[str]:
    request = urllib.request.Request(
        subscription_url(env, subscription_id),
        headers={
            "Host": required(env, "NOVA_DOMAIN"),
            "User-Agent": "v2rayN",
            "Accept": "*/*",
        },
    )
    with urllib.request.urlopen(request, timeout=15) as response:
        if response.status != 200:
            raise RuntimeError(f"subscription service returned HTTP {response.status}")
        return decode_subscription(response.read(4 * 1024 * 1024))


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


def query_one(query: dict[str, list[str]], key: str, default: str = "") -> str:
    values = query.get(key)
    return values[0] if values else default


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


def credential_field(protocol: str) -> str:
    return {"vless": "id", "trojan": "password", "hysteria": "auth"}[protocol]


def matched_client(inbound: dict[str, Any], credential: str) -> dict[str, Any]:
    field = credential_field(str(inbound["protocol"]).lower())
    clients = inbound.get("settings", {}).get("clients", [])
    for client in clients:
        if isinstance(client, dict) and str(client.get(field, "")) == credential:
            return client
    raise RuntimeError("subscription credential is not loaded in the Xray listener")


def split_alpn(value: str) -> list[str]:
    return [item.strip() for item in value.split(",") if item.strip()]


def tls_settings(
    inbound: dict[str, Any], query: dict[str, list[str]]
) -> dict[str, Any]:
    server = inbound.get("streamSettings", {}).get("tlsSettings", {})
    server_name = query_one(
        query, "sni", str(server.get("serverName", "")).strip()
    )
    if not server_name:
        raise RuntimeError("subscription TLS server name is empty")
    result: dict[str, Any] = {
        "serverName": server_name,
        "verifyPeerCertByName": query_one(query, "vcn", server_name),
        "fingerprint": query_one(query, "fp", "chrome"),
        "allowInsecure": False,
    }
    alpn = split_alpn(query_one(query, "alpn"))
    if alpn:
        result["alpn"] = alpn
    return result


def exact_outbound(
    inbound: dict[str, Any],
    parsed: urllib.parse.SplitResult,
    query: dict[str, list[str]],
    xray: pathlib.Path,
) -> dict[str, Any]:
    protocol = str(inbound["protocol"]).lower()
    credential = urllib.parse.unquote(parsed.username or "")
    client = matched_client(inbound, credential)
    runtime_port = int(inbound["port"])
    public_port = int(parsed.port or 0)
    transport = query_one(query, "type", "tcp").lower()
    security = query_one(query, "security").lower()
    runtime_stream = inbound.get("streamSettings", {})
    runtime_websocket = runtime_stream.get("wsSettings", {})
    websocket_path = urllib.parse.unquote(query_one(query, "path"))
    websocket_host = query_one(query, "host", parsed.hostname or "")
    managed_websocket = transport == "ws" and security == "tls"

    if managed_websocket:
        if (
            str(runtime_stream.get("network", "")).lower() != "ws"
            or str(runtime_stream.get("security", "")).lower() != "none"
            or str(runtime_websocket.get("path", "")) != websocket_path
        ):
            raise RuntimeError(
                "subscription WebSocket path does not match the managed listener"
            )
        if public_port != 443:
            raise RuntimeError("managed WebSocket subscription is not using TLS port 443")

    if protocol == "trojan":
        if security != "tls":
            raise RuntimeError("subscription Trojan security is not tls")
        if managed_websocket:
            return {
                "protocol": "trojan",
                "tag": "subscription-probe-out",
                "settings": {
                    "servers": [
                        {
                            "address": "127.0.0.1",
                            "port": public_port,
                            "password": credential,
                        }
                    ]
                },
                "streamSettings": {
                    "network": "ws",
                    "security": "tls",
                    "wsSettings": {
                        "path": websocket_path,
                        "host": websocket_host,
                    },
                    "tlsSettings": tls_settings(inbound, query),
                },
            }
        if transport != "tcp":
            raise RuntimeError("subscription Trojan transport is not tcp")
        return {
            "protocol": "trojan",
            "tag": "subscription-probe-out",
            "settings": {
                "servers": [
                    {
                        "address": "127.0.0.1",
                        "port": runtime_port,
                        "password": credential,
                    }
                ]
            },
            "streamSettings": {
                "network": "tcp",
                "security": "tls",
                "tcpSettings": {"header": {"type": "none"}},
                "tlsSettings": tls_settings(inbound, query),
            },
        }

    if protocol == "vless":
        if managed_websocket:
            flow = query_one(query, "flow")
            if flow:
                raise RuntimeError(
                    "managed VLESS WebSocket subscription must not include flow"
                )
            return {
                "protocol": "vless",
                "tag": "subscription-probe-out",
                "settings": {
                    "vnext": [
                        {
                            "address": "127.0.0.1",
                            "port": public_port,
                            "users": [
                                {
                                    "id": credential,
                                    "encryption": query_one(
                                        query, "encryption", "none"
                                    ),
                                }
                            ],
                        }
                    ]
                },
                "streamSettings": {
                    "network": "ws",
                    "security": "tls",
                    "wsSettings": {
                        "path": websocket_path,
                        "host": websocket_host,
                    },
                    "tlsSettings": tls_settings(inbound, query),
                },
            }
        if security != "reality":
            raise RuntimeError("subscription VLESS security is not reality")
        if transport != "tcp":
            raise RuntimeError("subscription VLESS transport is not tcp")
        reality = inbound.get("streamSettings", {}).get("realitySettings", {})
        private_key = str(reality.get("privateKey", ""))
        public_key = reality_public_key(xray, private_key)
        link_public_key = query_one(query, "pbk")
        if link_public_key != public_key:
            raise RuntimeError("subscription Reality public key does not match runtime")
        short_id = query_one(query, "sid")
        if short_id not in [str(value) for value in reality.get("shortIds", [])]:
            raise RuntimeError("subscription Reality short ID does not match runtime")
        server_name = query_one(query, "sni")
        if server_name not in [str(value) for value in reality.get("serverNames", [])]:
            raise RuntimeError("subscription Reality server name does not match runtime")
        user: dict[str, Any] = {
            "id": credential,
            "encryption": query_one(query, "encryption", "none"),
        }
        flow = query_one(query, "flow")
        if flow:
            if flow != str(client.get("flow", "")):
                raise RuntimeError("subscription VLESS flow does not match runtime")
            user["flow"] = flow
        return {
            "protocol": "vless",
            "tag": "subscription-probe-out",
            "settings": {
                "vnext": [
                    {
                        "address": "127.0.0.1",
                        "port": runtime_port,
                        "users": [user],
                    }
                ]
            },
            "streamSettings": {
                "network": "tcp",
                "security": "reality",
                "tcpSettings": {"header": {"type": "none"}},
                "realitySettings": {
                    "show": False,
                    "fingerprint": query_one(query, "fp", "chrome"),
                    "serverName": server_name,
                    "publicKey": link_public_key,
                    "shortId": short_id,
                    "spiderX": query_one(query, "spx", "/"),
                },
            },
        }

    if protocol == "hysteria":
        return {
            "protocol": "hysteria",
            "tag": "subscription-probe-out",
            "settings": {
                "address": "127.0.0.1",
                "port": runtime_port,
                "version": 2,
            },
            "streamSettings": {
                "network": "hysteria",
                "security": "tls",
                "tlsSettings": tls_settings(inbound, query),
                "hysteriaSettings": {
                    "version": 2,
                    "auth": credential,
                    "udpIdleTimeout": 60,
                },
            },
        }

    raise RuntimeError(f"unsupported protocol {protocol!r}")


def sanitized_log(log_path: pathlib.Path) -> str:
    if not log_path.exists():
        return "(no temporary Xray log)"
    lines = log_path.read_text(encoding="utf-8", errors="replace").splitlines()
    return "\n".join(lines[-12:]) or "(empty temporary Xray log)"


def probe_link(
    link: str,
    inbounds: list[dict[str, Any]],
    xray: pathlib.Path,
    workdir: pathlib.Path,
) -> tuple[str, int, bool, str] | None:
    parsed = urllib.parse.urlsplit(link)
    protocol = SCHEME_PROTOCOL.get(parsed.scheme.lower())
    if protocol is None:
        return None
    port = parsed.port
    if port is None:
        raise RuntimeError(f"{parsed.scheme} subscription link has no port")
    query = urllib.parse.parse_qs(parsed.query, keep_blank_values=True)
    transport = query_one(query, "type", "tcp").lower()
    subscription_path = urllib.parse.unquote(query_one(query, "path"))
    inbound = None
    for candidate in inbounds:
        if str(candidate.get("protocol", "")).lower() != protocol:
            continue
        stream = candidate.get("streamSettings", {})
        if (
            transport == "ws"
            and subscription_path
            and str(stream.get("network", "")).lower() == "ws"
            and str(stream.get("wsSettings", {}).get("path", ""))
            == subscription_path
        ):
            inbound = candidate
            break
        if transport != "ws" and int(candidate.get("port", 0)) == port:
            inbound = candidate
            break
    if inbound is None:
        return None
    proxy_port = free_tcp_port()
    config = {
        "log": {"loglevel": "warning"},
        "inbounds": [
            {
                "listen": "127.0.0.1",
                "port": proxy_port,
                "protocol": "http",
                "tag": "subscription-probe-http",
                "settings": {},
            }
        ],
        "outbounds": [exact_outbound(inbound, parsed, query, xray)],
    }
    config_path = workdir / f"{protocol}-{port}.json"
    log_path = workdir / f"{protocol}-{port}.log"
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
                return protocol, port, True, f"HTTP {status}"
            detail = result.stderr.strip() or f"curl exit {result.returncode}"
            return (
                protocol,
                port,
                False,
                f"{detail}; HTTP {status}\n{sanitized_log(log_path)}",
            )
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
    env = load_deploy_env()
    xray = find_xray()
    runtime = json.loads(CONFIG_PATH.read_text(encoding="utf-8"))
    inbounds = [
        item
        for item in runtime.get("inbounds", [])
        if str(item.get("tag", "")).startswith("commercial-in-")
        and str(item.get("protocol", "")).lower() in set(SCHEME_PROTOCOL.values())
    ]
    subscription_ids = active_subscription_ids(env)
    if not subscription_ids:
        fail("no active commercial subscriptions were found")
    print(f"Checking {len(subscription_ids)} active subscription(s), at most 10.")

    passed = 0
    failed = 0
    checked = 0
    with tempfile.TemporaryDirectory(prefix="nova-subscription-probe-") as directory:
        workdir = pathlib.Path(directory)
        for index, subscription_id in enumerate(subscription_ids, start=1):
            try:
                links = fetch_subscription(env, subscription_id)
            except Exception as exc:
                print(f"FAIL subscription-{index}: fetch/decode failed: {exc}")
                failed += 1
                continue
            seen: set[tuple[str, int]] = set()
            for link in links:
                try:
                    result = probe_link(link, inbounds, xray, workdir)
                    if result is None:
                        continue
                    protocol, port, ok, detail = result
                    if (protocol, port) in seen:
                        continue
                    seen.add((protocol, port))
                    checked += 1
                    label = "PASS" if ok else "FAIL"
                    print(
                        f"{label} subscription-{index} {protocol} port={port}: {detail}"
                    )
                    if ok:
                        passed += 1
                    else:
                        failed += 1
                except Exception as exc:
                    print(f"FAIL subscription-{index} managed link: {exc}")
                    checked += 1
                    failed += 1
            if not seen:
                print(f"FAIL subscription-{index}: no managed links were found")
                failed += 1
    print(f"SUMMARY: checked={checked} passed={passed} failed={failed}")
    return 0 if failed == 0 and checked > 0 else 2


if __name__ == "__main__":
    raise SystemExit(main())
