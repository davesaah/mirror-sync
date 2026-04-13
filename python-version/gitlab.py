import os
import requests
import sys
from dotenv import load_dotenv

load_dotenv()

# --- CONFIG ---
EXTERNAL_USER = "davesaah"
LOCAL_GITEA_URL = "https://git.davesaah-pc/api/v1"
LOCAL_TOKEN = os.getenv("LOCALHOST_TOKEN")
GITLAB_TOKEN = os.getenv("GITLAB_TOKEN")

VERIFY_CERT = "/etc/ssl/homelab/git.davesaah-pc.pem"


def create_gitlab_repo(repo_name, visibility):
    print(f"[*] Creating repo on GitLab: {repo_name}")

    url = "https://gitlab.com/api/v4/projects"
    headers = {
        "PRIVATE-TOKEN": GITLAB_TOKEN
    }

    payload = {
        "name": repo_name,
        "path": repo_name,
        "visibility": visibility  # public / private
    }

    resp = requests.post(url, json=payload, headers=headers)

    if resp.status_code == 201:
        print("[+] GitLab repo created")
        return True
    elif resp.status_code == 400 and "has already been taken" in resp.text:
        print("[*] Repo already exists on GitLab")
        return True
    else:
        print(f"[!] GitLab error: {resp.status_code} {resp.text}")
        return False


def add_gitlab_mirror(local_owner, repo_name):
    print("[*] Adding GitLab mirror...")

    endpoint = f"{LOCAL_GITEA_URL}/repos/{local_owner}/{repo_name}/push_mirrors"

    # Token embedded in URL (required)
    remote_url = f"https://oauth2:{GITLAB_TOKEN}@gitlab.com/{EXTERNAL_USER}/{repo_name}.git"

    payload = {
        "remote_address": remote_url,
        "sync_on_commit": True,
        "interval": "24h"
    }

    headers = {"Authorization": f"token {LOCAL_TOKEN}"}

    resp = requests.post(
        endpoint,
        json=payload,
        headers=headers,
        verify=VERIFY_CERT
    )

    print(f"[DEBUG] {resp.status_code} {resp.text}")

    if resp.status_code in [200, 201]:
        print("[+] GitLab mirror added successfully")
        return True
    else:
        print("[!] Failed to add GitLab mirror")
        return False


def get_gitea_description(owner, repo):
    print("[*] Fetching description from Gitea...")

    url = f"{LOCAL_GITEA_URL}/repos/{owner}/{repo}"
    headers = {"Authorization": f"token {LOCAL_TOKEN}"}

    resp = requests.get(url, headers=headers, verify=VERIFY_CERT)

    if resp.status_code != 200:
        print(f"[!] Failed to fetch Gitea repo: {resp.status_code} {resp.text}")
        return None

    return resp.json().get("description") or ""


def update_gitlab_description(repo, description):
    print("[*] Updating GitLab description...")

    if "[Mirror]" not in description:
        description = f"[Mirror] {description}"

    url = f"https://gitlab.com/api/v4/projects/{EXTERNAL_USER}%2F{repo}"
    headers = {
        "PRIVATE-TOKEN": GITLAB_TOKEN
    }

    payload = {
        "description": description.strip()
    }

    resp = requests.put(url, json=payload, headers=headers)

    if resp.status_code == 200:
        print("[+] GitLab description updated")
    else:
        print(f"[!] GitLab update failed: {resp.status_code} {resp.text}")


def main():
    if len(sys.argv) < 4:
        print("Usage: python script.py <local_owner> <repo_name> <public/private>")
        return

    local_owner = sys.argv[1]
    repo_name = sys.argv[2]
    visibility = sys.argv[3]

    if not GITLAB_TOKEN:
        print("[!] Missing GITLAB_TOKEN")
        return

    if create_gitlab_repo(repo_name, visibility):
        if add_gitlab_mirror(local_owner, repo_name):
            desc = get_gitea_description(local_owner, repo_name)

            if desc is not None:
                update_gitlab_description(repo_name, desc)


if __name__ == "__main__":
    main()
