import os
import requests
import sys
from dotenv import load_dotenv

load_dotenv()

# --- CONFIG ---
EXTERNAL_USER = "davesaah"
LOCAL_GITEA_URL = "https://git.davesaah-pc/api/v1"
LOCAL_TOKEN = os.getenv("LOCALHOST_TOKEN")
GITHUB_TOKEN = os.getenv("GITHUB_TOKEN")

VERIFY_CERT = "/etc/ssl/homelab/git.davesaah-pc.pem"


def create_github_repo(repo_name, visibility):
    print(f"[*] Creating repo on GitHub: {repo_name}")

    url = "https://api.github.com/user/repos"
    headers = {
        "Authorization": f"token {GITHUB_TOKEN}",
        "Accept": "application/vnd.github+json"
    }

    payload = {
        "name": repo_name,
        "private": visibility == "private"
    }

    resp = requests.post(url, json=payload, headers=headers)

    if resp.status_code == 201:
        print("[+] GitHub repo created")
        return True
    elif resp.status_code == 422:
        print("[*] Repo already exists on GitHub")
        return True
    else:
        print(f"[!] GitHub error: {resp.status_code} {resp.text}")
        return False


def add_github_mirror(local_owner, repo_name):
    print("[*] Adding GitHub mirror...")

    endpoint = f"{LOCAL_GITEA_URL}/repos/{local_owner}/{repo_name}/push_mirrors"

    remote_url = f"https://{EXTERNAL_USER}:{GITHUB_TOKEN}@github.com/{EXTERNAL_USER}/{repo_name}.git"

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
        print("[+] GitHub mirror added successfully")
        return True
    else:
        print("[!] Failed to add GitHub mirror")
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


def update_github_description(repo, description):
    print("[*] Updating GitHub description...")

    # Avoid duplicate suffix
    if "[Mirror]" not in description:
        description = f"[Mirror] {description}"

    url = f"https://api.github.com/repos/{EXTERNAL_USER}/{repo}"
    headers = {
        "Authorization": f"token {GITHUB_TOKEN}",
        "Accept": "application/vnd.github+json"
    }

    payload = {"description": description.strip()}

    resp = requests.patch(url, json=payload, headers=headers)

    if resp.status_code == 200:
        print("[+] GitHub description updated")
    else:
        print(f"[!] GitHub update failed: {resp.status_code} {resp.text}")


def main():
    if len(sys.argv) < 4:
        print("Usage: python script.py <local_owner> <repo_name> <public/private>")
        return

    local_owner = sys.argv[1]
    repo_name = sys.argv[2]
    visibility = sys.argv[3]

    if not GITHUB_TOKEN:
        print("[!] Missing GITHUB_TOKEN")
        return

    if create_github_repo(repo_name, visibility):
        if add_github_mirror(local_owner, repo_name):
            desc = get_gitea_description(local_owner, repo_name)

            if desc is not None:
                update_github_description(repo_name, desc)


if __name__ == "__main__":
    main()
