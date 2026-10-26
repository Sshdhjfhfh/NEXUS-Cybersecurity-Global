"""Inspect and answer live session-relay requests without reusing expired work."""
import argparse
import json
from pathlib import Path
import time
import uuid
import os

from session_bridge import DEFAULT_RESPONSE_SECONDS, validate_message


def live_requests(queue, now=None):
    now = time.time() if now is None else now
    for path in sorted(queue.glob("*.request.json"), key=lambda p: p.stat().st_mtime):
        item = json.loads(path.read_text())
        ident = item["id"]
        if path.name != ident + ".request.json":
            continue
        if (queue / (ident + ".done.json")).exists() or (queue / (ident + ".response.json")).exists():
            continue
        expires = item.get("expires_at", item["received_at"] + DEFAULT_RESPONSE_SECONDS)
        if now < expires:
            yield item, expires - now


def submit(queue, ident, message):
    if len(ident) != 32 or any(ch not in "0123456789abcdef" for ch in ident):
        raise ValueError("invalid request ID")
    current = next((item for item, _ in live_requests(queue) if item["id"] == ident), None)
    if current is None:
        raise ValueError("request is expired, finished, answered, or missing")
    validate_message(message, current["request"])
    destination = queue / (ident + ".response.json")
    temporary = queue / (ident + "." + uuid.uuid4().hex + ".tmp")
    try:
        fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, "w") as f:
            json.dump(message, f, ensure_ascii=False)
        # Recheck after serialization; the bridge also enforces its own deadline.
        if not any(item["id"] == ident for item, _ in live_requests(queue)):
            raise ValueError("request is no longer live")
        os.link(temporary, destination)  # atomic, never replaces another reply
    finally:
        temporary.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("queue", type=Path)
    parser.add_argument("--reply", metavar="REQUEST_ID")
    parser.add_argument("--message", type=Path, help="JSON assistant message to publish")
    args = parser.parse_args()
    if bool(args.reply) != bool(args.message):
        parser.error("--reply and --message are required together")
    if args.reply:
        submit(args.queue, args.reply, json.loads(args.message.read_text()))
        print(json.dumps({"id": args.reply, "status": "submitted"}))
    else:
        for item, remaining in live_requests(args.queue):
            request = item["request"]
            print(json.dumps({"id": item["id"], "remaining_seconds": round(remaining, 1),
                              "messages": len(request.get("messages", [])),
                              "tools": [t["function"]["name"] for t in request.get("tools", [])]}))


if __name__ == "__main__":
    main()
