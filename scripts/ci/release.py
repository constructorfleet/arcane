#!/usr/bin/env python3
"""Validate and publish Constructorfleet releases using the repository job token."""
import argparse
from datetime import datetime, timezone
import json
import os
import re
import subprocess
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

VERSION = re.compile(r"(\d+)\.(\d+)\.(\d+)-cf\.(\d+)")
DIGEST = re.compile(r"sha256:[0-9a-f]{64}")


def metadata(text):
    values = dict(line.split("=", 1) for line in text.splitlines() if line and not line.startswith("#"))
    if not VERSION.fullmatch(values.get("VERSION", "")):
        raise ValueError("VERSION must use X.Y.Z-cf.N")
    for key in ("REGISTRY", "REPOSITORY", "PLATFORMS"):
        if not values.get(key):
            raise ValueError(f"Missing {key}")
    return values


def validate(current, previous, notes):
    version = current["VERSION"]
    old = tuple(map(int, VERSION.fullmatch(previous["VERSION"]).groups()))
    new = tuple(map(int, VERSION.fullmatch(version).groups()))
    if not ((new[:3] == old[:3] and new[3] == old[3] + 1) or (new[:3] > old[:3] and new[3] == 1)):
        raise ValueError("Advance the fork counter exactly once, or use cf.1 for a newer upstream base")
    if not notes.startswith(f"# {version}\n") or len(notes.strip().splitlines()) < 4:
        raise ValueError("RELEASE_NOTES.md must start with the new version and describe the changes")


def api(method, path, payload=None, missing=False):
    root = os.environ["GITEA_SERVER_URL"].rstrip("/")
    repo = os.environ["GITEA_REPOSITORY"]
    request = urllib.request.Request(
        f"{root}/api/v1/repos/{repo}/{path}",
        data=None if payload is None else json.dumps(payload).encode(),
        method=method,
        headers={"Authorization": "token " + os.environ["GITEA_TOKEN"], "Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            raw = response.read()
            return json.loads(raw) if raw else None
    except urllib.error.HTTPError as error:
        if error.code == 404 and missing:
            return None
        raise RuntimeError(f"Gitea {method} {path}: HTTP {error.code}") from None


def release_body(values, notes, sha, digests, pull, server, repository):
    lines = [notes.strip(), "", f"Source: [{sha[:12]}]({server}/{repository}/commit/{sha})"]
    if pull:
        lines.append(f"Pull request: [#{pull['number']}]({pull['html_url']})")
    lines.extend(["", "## Container images", "", "| Image | Version | Immutable reference | Harbor |", "| --- | --- | --- | --- |"])
    for image, digest in digests.items():
        if not DIGEST.fullmatch(digest):
            raise ValueError(f"Invalid {image} digest")
        name = f"{values['REPOSITORY']}-{image}"
        ref = f"{values['REGISTRY']}/{name}"
        # Harbor project ID is explicit deployment metadata, never inferred from a display name.
        repo_name = urllib.parse.quote(name.split("/", 1)[1], safe="")
        harbor = f"https://{values['REGISTRY']}/harbor/projects/{values['HARBOR_PROJECT_ID']}/repositories/{repo_name}/artifacts/{digest}"
        lines.append(f"| {image} | `{ref}:{values['VERSION']}` | `{ref}@{digest}` | [Artifact]({harbor}) |")
    lines.extend(["", f"Platforms: `{values['PLATFORMS']}`. Build provenance is attached to the images.", "", "Both images were built before any latest promotion. SwarmCD owns deployment; a published release does not by itself prove runtime acceptance.", "", "Validation and limitations are recorded above. Operator documentation: [Arcane in Docmost](https://docmost.home.prettybaked.com/s/general/p/arcane-hkk6Tx4CHx)."])
    return "\n".join(lines) + "\n"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=["check", "metadata", "draft", "publish", "promotion"])
    parser.add_argument("--base")
    parser.add_argument("--manager-digest")
    parser.add_argument("--agent-digest")
    args = parser.parse_args()
    values = metadata(Path("RELEASE").read_text())
    notes = Path("RELEASE_NOTES.md").read_text()
    if args.command == "check":
        previous = metadata(subprocess.check_output(["git", "show", f"{args.base}:RELEASE"], text=True))
        validate(values, previous, notes)
        print("Release contract passed")
        return
    sha = os.environ["GITEA_SHA"]
    tag = "v" + values["VERSION"]
    existing_tag = api("GET", "tags/" + tag, missing=True)
    if existing_tag and existing_tag["commit"]["sha"] != sha:
        raise ValueError(f"{tag} already belongs to another commit; never overwrite releases")
    existing = api("GET", "releases/tags/" + tag, missing=True)
    if existing and existing["target_commitish"] != sha:
        raise ValueError("Existing release points to another commit")
    if args.command == "metadata":
        with open(os.environ["GITHUB_OUTPUT"], "a") as output:
            output.write("published=" + str(bool(existing and not existing["draft"])).lower() + "\n")
            output.write("build_time=" + datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ") + "\n")
            for key in ("REGISTRY", "REPOSITORY", "VERSION", "PLATFORMS", "PROVENANCE"):
                output.write(f"{key.lower()}={values[key]}\n")
        return
    if args.command == "promotion":
        head = api("GET", "commits/main")
        with open(os.environ["GITHUB_OUTPUT"], "a") as output:
            output.write("latest=" + str(head["sha"] == sha).lower() + "\n")
        return
    if args.command == "publish":
        if not existing:
            raise ValueError("Draft release missing")
        api("PATCH", f"releases/{existing['id']}", {"draft": False})
        print(f"Published {tag}")
        return
    if existing and not existing["draft"]:
        print(f"{tag} is already published; leaving release and images unchanged")
        return
    pull = api("GET", f"commits/{sha}/pull", missing=True)
    if not pull or not pull.get("merged") or pull.get("merge_commit_sha") != sha:
        raise ValueError("Release source must be a merged pull request")
    body = release_body(values, notes, sha, {"manager": args.manager_digest or "", "agent": args.agent_digest or ""}, pull,
                        os.environ["GITEA_SERVER_URL"].rstrip("/"), os.environ["GITEA_REPOSITORY"])
    Path("release-notes.generated.md").write_text(body)
    payload = {"tag_name": tag, "target_commitish": sha, "name": values["VERSION"], "body": body, "draft": True, "prerelease": True}
    if existing:
        # A successful retry must not hide an already published release.
        payload["draft"] = existing["draft"]
        api("PATCH", f"releases/{existing['id']}", payload)
    else:
        api("POST", "releases", payload)
    print(f"Prepared {tag}")


if __name__ == "__main__":
    main()
