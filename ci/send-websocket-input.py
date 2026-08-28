#!/usr/bin/env python3
"""Send terminal input frames to the iSH WebSocket transport.

This is used only by the Android smoke test. It speaks the small RFC 6455
client handshake/frame subset needed by WebChannelServer, avoiding a third-
party dependency on GitHub Actions runners.
"""

from __future__ import annotations

import base64
import hashlib
import json
import os
import socket
import sys
import time


def send_text(sock: socket.socket, value: str) -> None:
    payload = json.dumps({"input": value}, separators=(",", ":")).encode("utf-8")
    if len(payload) >= 126:
        raise ValueError("smoke input frame is unexpectedly large")
    mask = os.urandom(4)
    masked = bytes(byte ^ mask[index % 4] for index, byte in enumerate(payload))
    sock.sendall(bytes((0x81, 0x80 | len(payload))) + mask + masked)


def close_socket(sock: socket.socket) -> None:
    try:
        mask = os.urandom(4)
        sock.sendall(bytes((0x88, 0x80)) + mask)
    except OSError:
        pass
    sock.close()


def main() -> int:
    if len(sys.argv) != 5:
        print(
            "usage: send-websocket-input.py PORT FIRST_COMMAND "
            "SECOND_COMMAND WAIT_SECONDS",
            file=sys.stderr,
        )
        return 2
    port = int(sys.argv[1])
    first_command = sys.argv[2]
    second_command = sys.argv[3]
    wait_seconds = float(sys.argv[4])

    key = base64.b64encode(os.urandom(16)).decode("ascii")
    request = (
        f"GET / HTTP/1.1\r\n"
        f"Host: 127.0.0.1:{port}\r\n"
        "Upgrade: websocket\r\n"
        "Connection: Upgrade\r\n"
        f"Sec-WebSocket-Key: {key}\r\n"
        "Sec-WebSocket-Version: 13\r\n\r\n"
    ).encode("ascii")
    expected_accept = base64.b64encode(
        hashlib.sha1((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode("ascii")).digest()
    ).decode("ascii")

    with socket.create_connection(("127.0.0.1", port), timeout=10) as sock:
        sock.sendall(request)
        response = b""
        while b"\r\n\r\n" not in response:
            chunk = sock.recv(4096)
            if not chunk:
                raise RuntimeError("WebSocket handshake closed before response")
            response += chunk
            if len(response) > 16384:
                raise RuntimeError("WebSocket handshake response is too large")
        header = response.split(b"\r\n\r\n", 1)[0].decode("latin1")
        if " 101 " not in header or f"Sec-WebSocket-Accept: {expected_accept}" not in header:
            raise RuntimeError(f"WebSocket handshake failed: {header!r}")

        send_text(sock, first_command + "\r")
        print(f"Sent WebSocket command: {first_command}", flush=True)
        time.sleep(wait_seconds)
        send_text(sock, second_command + "\r")
        print(f"Sent WebSocket command: {second_command}", flush=True)
        time.sleep(2)
        close_socket(sock)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

