#!/usr/bin/env python3
"""E2E smoke tests for kubectl-debug-queries.

Runs a suite of tests against every CLI command using resources common
to all OpenShift clusters.  Build the binary first (e.g. ``make e2e``
or ``make build && python3 tests/e2e_smoke.py``).

Usage:
    make e2e
"""

import json
import os
import subprocess
import sys
from typing import List, Optional

BINARY = os.path.join(os.path.dirname(__file__), "..", "kubectl-debug-queries")

passed = 0
failed = 0
errors = []  # type: List[str]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def run(args, expect_fail=False):
    """Run the binary with the given args, return (stdout, stderr, returncode)."""
    try:
        result = subprocess.run(
            [BINARY] + args,
            capture_output=True,
            text=True,
            timeout=60,
        )
        return result.stdout, result.stderr, result.returncode
    except subprocess.TimeoutExpired as exc:
        stdout = exc.stdout or ""
        stderr = (exc.stderr or "") + "\n[TIMEOUT] command timed out after 60s"
        return stdout, stderr, 1


def record(name: str, ok: bool, detail: str = ""):
    global passed, failed
    if ok:
        passed += 1
        print(f"  PASS  {name}")
    else:
        failed += 1
        msg = f"  FAIL  {name}"
        if detail:
            msg += f"  -- {detail}"
        print(msg)
        errors.append(name)


def assert_exit_ok(name: str, rc: int, stderr: str = "") -> bool:
    ok = rc == 0
    record(name, ok, f"exit={rc} stderr={stderr[:120]}" if not ok else "")
    return ok


def assert_exit_fail(name: str, rc: int) -> bool:
    ok = rc != 0
    record(name, ok, "expected non-zero exit code" if not ok else "")
    return ok


def assert_contains(name: str, text: str, substring: str) -> bool:
    ok = substring in text
    record(name, ok, f"output missing '{substring}'" if not ok else "")
    return ok


def parse_json(text: str):
    try:
        return json.loads(text), None
    except json.JSONDecodeError as exc:
        return None, str(exc)


def assert_valid_json(name: str, text: str):
    data, err = parse_json(text)
    record(name, err is None, f"invalid JSON: {err}" if err else "")
    return data


def assert_json_array_max_length(name: str, text: str, max_len: int):
    data, err = parse_json(text)
    if err:
        record(name, False, f"invalid JSON: {err}")
        return None
    if not isinstance(data, list):
        record(name, False, "expected JSON array")
        return None
    ok = len(data) <= max_len
    record(name, ok, f"got {len(data)} items, expected <= {max_len}" if not ok else "")
    return data


# ---------------------------------------------------------------------------
# Tests: version
# ---------------------------------------------------------------------------

def test_version():
    print("[version]")
    stdout, stderr, rc = run(["version"])
    if assert_exit_ok("version exits 0", rc, stderr):
        assert_contains("version output", stdout, "kubectl-debug-queries")


# ---------------------------------------------------------------------------
# Tests: list
# ---------------------------------------------------------------------------

def test_list():
    print("[list]")

    # 1. List pods in openshift-apiserver (markdown)
    stdout, stderr, rc = run(["list", "--resource", "pods", "--namespace", "openshift-apiserver"])
    if assert_exit_ok("list pods openshift-apiserver", rc, stderr):
        assert_contains("list pods contains apiserver", stdout, "apiserver")

    # 2. List pods in openshift-dns (json)
    stdout, stderr, rc = run(["list", "--resource", "pods", "--namespace", "openshift-dns", "--output", "json"])
    if assert_exit_ok("list pods openshift-dns json", rc, stderr):
        data = assert_valid_json("list pods dns valid json", stdout)
        if data and isinstance(data, list) and len(data) > 0:
            assert_contains("list pods dns has name field", json.dumps(data[0]), "name")

    # 3. --limit flag with JSON output
    stdout, stderr, rc = run([
        "list", "--resource", "pods", "--namespace", "openshift-monitoring",
        "--limit", "5", "--output", "json",
    ])
    if assert_exit_ok("list pods monitoring --limit 5 json", rc, stderr):
        assert_json_array_max_length("list pods monitoring --limit 5 <=5 items", stdout, 5)

    # 3b. --limit flag with markdown output
    stdout, stderr, rc = run([
        "list", "--resource", "pods", "--namespace", "openshift-monitoring",
        "--limit", "3",
    ])
    if assert_exit_ok("list pods monitoring --limit 3 markdown", rc, stderr):
        rows = [line for line in stdout.strip().splitlines() if line.strip() and not line.startswith("| ---")]
        table_rows = rows[1:]  # skip header
        record("list pods monitoring --limit 3 <=3 rows", len(table_rows) <= 3,
               f"got {len(table_rows)} rows, expected <= 3" if len(table_rows) > 3 else "")

    # 4. List pods with query filter
    stdout, stderr, rc = run([
        "list", "--resource", "pods", "--namespace", "openshift-dns",
        "--output", "json",
        "--query", "select name, status.phase where status.phase = 'Running'",
    ])
    if assert_exit_ok("list pods dns query Running", rc, stderr):
        data, _ = parse_json(stdout)
        if data and isinstance(data, list):
            all_running = all(
                item.get("status.phase") == "Running"
                for item in data
            )
            record("list pods dns all Running", all_running,
                   "not all pods are Running" if not all_running else "")

    # 5. List nodes (all-namespaces)
    stdout, stderr, rc = run(["list", "--resource", "nodes", "--all-namespaces"])
    assert_exit_ok("list nodes", rc, stderr)

    # 6. List namespaces
    stdout, stderr, rc = run(["list", "--resource", "namespaces", "--all-namespaces"])
    if assert_exit_ok("list namespaces", rc, stderr):
        assert_contains("list namespaces has openshift-apiserver", stdout, "openshift-apiserver")

    # 7. List deployments in openshift-apiserver
    stdout, stderr, rc = run(["list", "--resource", "deployments", "--namespace", "openshift-apiserver"])
    if assert_exit_ok("list deployments openshift-apiserver", rc, stderr):
        assert_contains("list deployments has apiserver", stdout, "apiserver")

    # 8. Positional arg: list pods
    stdout, stderr, rc = run(["list", "pods", "--namespace", "openshift-dns"])
    if assert_exit_ok("list positional arg pods", rc, stderr):
        assert_contains("list positional has dns-default", stdout, "dns-default")


# ---------------------------------------------------------------------------
# Tests: get
# ---------------------------------------------------------------------------

def _first_pod_name(namespace):
    # type: (str) -> Optional[str]
    """Dynamically fetch the first pod name in a namespace."""
    stdout, _, rc = run([
        "list", "--resource", "pods", "--namespace", namespace,
        "--output", "json", "--query", "select name limit 1",
    ])
    if rc != 0:
        return None
    data, _ = parse_json(stdout)
    if data and isinstance(data, list) and len(data) > 0:
        return data[0].get("name")
    return None


def test_get():
    print("[get]")

    pod_name = _first_pod_name("openshift-apiserver")
    if not pod_name:
        record("get: discover pod name", False, "could not find a pod in openshift-apiserver")
        return

    # 1. Get a specific pod
    stdout, stderr, rc = run([
        "get", "--resource", "pod", "--name", pod_name,
        "--namespace", "openshift-apiserver",
    ])
    if assert_exit_ok("get pod by name", rc, stderr):
        assert_contains("get pod output has name", stdout, pod_name)

    # 2. Get deployment
    stdout, stderr, rc = run([
        "get", "--resource", "deployment", "--name", "apiserver",
        "--namespace", "openshift-apiserver",
    ])
    assert_exit_ok("get deployment apiserver", rc, stderr)

    # 3. Get with json output
    stdout, stderr, rc = run([
        "get", "--resource", "deployment", "--name", "apiserver",
        "--namespace", "openshift-apiserver", "--output", "json",
    ])
    if assert_exit_ok("get deployment json", rc, stderr):
        assert_valid_json("get deployment valid json", stdout)

    # 4. Get with query select
    stdout, stderr, rc = run([
        "get", "--resource", "pod", "--name", pod_name,
        "--namespace", "openshift-apiserver",
        "--output", "json", "--query", "select name",
    ])
    if assert_exit_ok("get pod select name", rc, stderr):
        data = assert_valid_json("get pod select name json", stdout)
        if data and isinstance(data, dict):
            record("get pod select only name field", "name" in data,
                   f"got keys: {list(data.keys())}" if "name" not in data else "")

    # 5. Positional args: get pod <name>
    stdout, stderr, rc = run([
        "get", "pod", pod_name,
        "--namespace", "openshift-apiserver",
    ])
    if assert_exit_ok("get positional args", rc, stderr):
        assert_contains("get positional has pod name", stdout, pod_name)


# ---------------------------------------------------------------------------
# Tests: logs
# ---------------------------------------------------------------------------

def test_logs():
    print("[logs]")

    # 1. Logs from deployment/apiserver with tail
    stdout, stderr, rc = run([
        "logs", "--name", "deployment/apiserver",
        "--namespace", "openshift-apiserver", "--tail", "10",
    ])
    if assert_exit_ok("logs deployment/apiserver tail=10", rc, stderr):
        record("logs output non-empty", len(stdout.strip()) > 0,
               "empty output" if not stdout.strip() else "")

    # 2. Logs with json output
    stdout, stderr, rc = run([
        "logs", "--name", "deployment/apiserver",
        "--namespace", "openshift-apiserver",
        "--tail", "5", "--output", "json",
    ])
    if assert_exit_ok("logs json output", rc, stderr):
        data = assert_valid_json("logs valid json", stdout)
        if data and isinstance(data, list) and len(data) > 0:
            first = data[0]
            has_fields = "message" in first or "raw_line" in first
            record("logs json has expected fields", has_fields,
                   f"got keys: {list(first.keys())}" if not has_fields else "")

    # 3. Logs with raw output
    stdout, stderr, rc = run([
        "logs", "--name", "deployment/apiserver",
        "--namespace", "openshift-apiserver",
        "--tail", "5", "--output", "raw",
    ])
    if assert_exit_ok("logs raw output", rc, stderr):
        record("logs raw non-empty", len(stdout.strip()) > 0,
               "empty output" if not stdout.strip() else "")

    # 4. Logs with sort-by time_desc
    stdout, stderr, rc = run([
        "logs", "--name", "deployment/apiserver",
        "--namespace", "openshift-apiserver",
        "--tail", "10", "--sort-by", "time_desc",
    ])
    assert_exit_ok("logs sort-by time_desc", rc, stderr)


# ---------------------------------------------------------------------------
# Tests: events
# ---------------------------------------------------------------------------

def test_events():
    print("[events]")

    # 1. Events in default namespace (may be empty, should still succeed)
    stdout, stderr, rc = run(["events", "--namespace", "default"])
    assert_exit_ok("events default namespace", rc, stderr)

    # 2. Events all-namespaces with limit
    stdout, stderr, rc = run(["events", "--all-namespaces", "--limit", "5"])
    assert_exit_ok("events all-namespaces limit=5", rc, stderr)

    # 3. Events json output with --limit
    stdout, stderr, rc = run([
        "events", "--all-namespaces", "--limit", "5", "--output", "json",
    ])
    if assert_exit_ok("events json output", rc, stderr):
        data = assert_valid_json("events valid json", stdout)
        if data and isinstance(data, list):
            record("events --limit 5 json <=5 items", len(data) <= 5,
                   f"got {len(data)} items, expected <= 5" if len(data) > 5 else "")

    # 4. Events with query filter
    stdout, stderr, rc = run([
        "events", "--all-namespaces", "--limit", "5",
        "--query", "where type = 'Normal'",
    ])
    assert_exit_ok("events query Normal", rc, stderr)


# ---------------------------------------------------------------------------
# Tests: error handling
# ---------------------------------------------------------------------------

def test_errors():
    print("[error handling]")

    # 1. list with no --resource
    _, _, rc = run(["list", "--namespace", "default"])
    assert_exit_fail("list missing --resource", rc)

    # 2. get with missing --name
    _, _, rc = run(["get", "--resource", "pod", "--namespace", "default"])
    assert_exit_fail("get missing --name", rc)

    # 3. logs with missing --name
    _, _, rc = run(["logs", "--namespace", "default"])
    assert_exit_fail("logs missing --name", rc)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    if not os.path.isfile(BINARY):
        print(f"Binary not found: {BINARY}")
        print("Run 'make build' first, or use 'make e2e'.")
        sys.exit(2)

    print("=" * 60)
    print("E2E Smoke Tests")
    print("=" * 60)

    test_version()
    test_list()
    test_get()
    test_logs()
    test_events()
    test_errors()

    print("=" * 60)
    print(f"Results: {passed} passed, {failed} failed")
    if errors:
        print("Failed tests:")
        for name in errors:
            print(f"  - {name}")
    print("=" * 60)

    sys.exit(1 if failed > 0 else 0)


if __name__ == "__main__":
    main()
