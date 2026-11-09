#!/usr/bin/env python3
"""End-to-end smoke test for locally running FinLink API and sync worker.

Requires mock-bank infrastructure, applied migrations, API and worker to be running.
Uses only the Python standard library and creates isolated mock connections.
"""
import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

BASE = os.environ.get("FINLINK_API_BASE_URL", "http://127.0.0.1:8080").rstrip("/")
TIMEOUT = int(os.environ.get("FINLINK_SMOKE_TIMEOUT_SECONDS", "60"))


def request(method, path, payload=None, headers=None, expected=200):
    data = json.dumps(payload).encode() if payload is not None else None
    req = urllib.request.Request(
        BASE + path, data=data, method=method,
        headers={"Content-Type": "application/json", **(headers or {})},
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            body = json.load(response)
            assert response.status == expected, (path, response.status, body)
            assert response.headers.get("Cache-Control") == "no-store", path
            return body
    except urllib.error.HTTPError as error:
        raise AssertionError(
            f"{method} {path}: expected {expected}, got {error.code}: "
            f"{error.read().decode()}"
        ) from error


def main():
    assert request("GET", "/health")["status"] == "ok"
    for provider in ("northstar", "harbor"):
        connection = request(
            "POST", "/v1/connections", {"provider": provider, "credentials": (
                {"username": "smoke-" + uuid.uuid4().hex, "password": "mock-only"}
                if provider == "northstar" else {"token": "smoke-" + uuid.uuid4().hex}
            )},
            expected=201,
        )
        connection_id = str(connection["id"])
        path = "/v1/connections/" + urllib.parse.quote(connection_id, safe="")
        restored = request("GET", path)
        assert str(restored["id"]) == connection_id, (provider, restored)

        key = "finlink-smoke-" + str(uuid.uuid4())
        run = request(
            "POST", path + "/sync", headers={"Idempotency-Key": key},
            expected=202,
        )
        replay = request(
            "POST", path + "/sync", headers={"Idempotency-Key": key},
            expected=202,
        )
        assert run["id"] == replay["id"], (run, replay)

        deadline = time.monotonic() + TIMEOUT
        while True:
            current = request("GET", "/v1/sync-runs/" + str(run["id"]))
            status = current["status"]
            if status == "succeeded":
                break
            if status == "failed":
                raise AssertionError(f"{provider} sync failed: {current}")
            if time.monotonic() >= deadline:
                raise TimeoutError(f"{provider} sync timed out: {current}")
            time.sleep(1)

        accounts = request("GET", path + "/accounts")["accounts"]
        assert accounts, f"{provider} returned no accounts"
        assert current.get("account_count", 0) > 0, (provider, current)
        assert current.get("transaction_count", 0) > 0, (provider, current)
        seen = set()
        offset = 0
        while True:
            page = request("GET", path + f"/transactions?limit=1&offset={offset}")
            transactions = page["transactions"]
            assert len(transactions) <= 1, (provider, offset, page)
            assert page["limit"] == 1, (provider, page)
            assert page["offset"] == offset, (provider, offset, page)
            for transaction in transactions:
                # Canonical transactions expose provider_external_id and fingerprint,
                # not a database-generated id. Match the identity used for deduplication.
                external_id = transaction.get("provider_external_id")
                fingerprint = transaction.get("fingerprint")
                assert external_id or fingerprint, f"{provider} transaction has no identity: {transaction}"
                txid = (str(transaction["connection_id"]), str(external_id or fingerprint))
                assert txid[0] == connection_id, f"{provider} returned a foreign transaction: {transaction}"
                assert txid not in seen, f"duplicate transaction: {txid}"
                seen.add(txid)
            offset += len(transactions)
            if not page["may_have_more"]:
                break
            assert transactions, f"{provider} pagination stalled"
            assert offset <= 1000000, "pagination exceeded API maximum offset"
        assert seen, f"{provider} returned no transactions"
        # A page immediately beyond the last record must be empty and terminal.
        exhausted = request("GET", path + f"/transactions?limit=1&offset={offset}")
        assert exhausted["transactions"] == [], (provider, exhausted)
        assert exhausted["may_have_more"] is False, (provider, exhausted)
        assert exhausted["limit"] == 1 and exhausted["offset"] == offset, (provider, exhausted)
        # Re-reading the same connection must not change the visible financial data.
        repeat = request("GET", path + "/transactions?limit=100&offset=0")
        assert len(repeat["transactions"]) == len(seen), (provider, repeat)
        print(f"PASS {provider}: {len(accounts)} accounts, {len(seen)} transactions")

    print("PASS mock-bank end-to-end smoke test")


if __name__ == "__main__":
    main()
