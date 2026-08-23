#!/usr/bin/env python3
"""API smoke in mock/offline mode. Cost: ¥0."""
import json
import os
import time
import urllib.error
import urllib.request

API = os.environ.get("API_BASE", "http://api:8080").rstrip("/")


def req(method, path, body=None, token=None, expected=200):
    data = None if body is None else json.dumps(body).encode()
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    r = urllib.request.Request(API + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=20) as resp:
            raw = resp.read()
            code = resp.status
            payload = json.loads(raw.decode() or "{}")
    except urllib.error.HTTPError as e:
        code = e.code
        payload = json.loads(e.read().decode() or "{}")
    assert code == expected, f"{method} {path} => {code} {payload} want {expected}"
    return payload


def main():
    h = req("GET", "/healthz")
    assert h.get("ok") is True
    print("[PASS] Health Check")

    login = req("POST", "/api/v1/auth/login", {"email": "owner@lumen.local", "password": "Owner123!"})
    token = login["data"]["tokens"]["access_token"]
    print("[PASS] Auth")

    me = req("GET", "/api/v1/me", token=token)
    tenant = me["data"]["tenant"] or {}
    assert tenant.get("name") or tenant.get("Name")
    print("[PASS] Tenant isolation envelope")

    lists = req("GET", "/api/v1/lists", token=token)["data"]
    assert lists
    print("[PASS] Lists")

    camps = req("GET", "/api/v1/campaigns", token=token)
    print("[PASS] Campaigns list")

    pipe = req("GET", "/api/v1/pipeline/stats", token=token)
    assert "data" in pipe
    print("[PASS] Pipeline stats")

    # viewer cannot write
    v = req("POST", "/api/v1/auth/login", {"email": "viewer@lumen.local", "password": "Viewer123!"})
    vt = v["data"]["tokens"]["access_token"]
    req("POST", "/api/v1/lists", {"name": "x"}, token=vt, expected=403)
    print("[PASS] Viewer write forbidden")

    print("[PASS] Mock/offline smoke complete Cost=¥0")


if __name__ == "__main__":
    for i in range(30):
        try:
            req("GET", "/healthz")
            break
        except Exception:
            time.sleep(2)
    else:
        raise SystemExit("api not ready")
    main()
