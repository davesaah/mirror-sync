import os
import requests
import sys
from dotenv import load_dotenv

load_dotenv()

# --- CONFIG ---
EXTERNAL_USER = "davesaah"
LOCAL_GITEA_URL = "https://git.davesaah-pc/api/v1"
LOCAL_TOKEN = os.getenv("LOCALHOST_TOKEN")
CODEBERG_TOKEN = os.getenv("CODEBERG_TOKEN")
VERIFY_CERT = "/etc/ssl/homelab/git.davesaah-pc.pem"


def create_codeberg_repo(service_name, repo_name, token, base_api_url, visibility="private"):
    url = f"{base_api_url}/user/repos"
    headers = {"Authorization": f"token {token}"}
    payload = {"name": repo_name, "private": visibility == "private"}

    resp = requests.post(url, json=payload, headers=headers)
    if resp.status_code == 201:
        print(f"[+] {service_name} repo created")
        return True
    elif resp.status_code == 422 and "already exists" in resp.text:
        print(f"[*] Repo already exists on {service_name}")
        return True
    else:
        print(f"[!] {service_name} error: {resp.status_code} {resp.text}")
        return False


def add_gitea_mirror(local_owner, repo_name, service_name, remote_url):
    print(f"[*] Adding {service_name} mirror...")

    endpoint = f"{LOCAL_GITEA_URL}/repos/{local_owner}/{repo_name}/push_mirrors"
    payload = {
        "remote_address": remote_url,
        "sync_on_commit": True,
        "interval": "24h"
    }
    headers = {"Authorization": f"token {LOCAL_TOKEN}"}

    resp = requests.post(endpoint, json=payload, headers=headers, verify=VERIFY_CERT)
    print(f"[DEBUG] {resp.status_code} {resp.text}")

    if resp.status_code in [200, 201]:
        print(f"[+] {service_name} mirror added successfully")
        return True
    else:
        print(f"[!] Failed to add {service_name} mirror")
        return False


def get_gitea_description(owner, repo):
    url = f"{LOCAL_GITEA_URL}/repos/{owner}/{repo}"
    headers = {"Authorization": f"token {LOCAL_TOKEN}"}
    resp = requests.get(url, headers=headers, verify=VERIFY_CERT)
    if resp.status_code != 200:
        print(f"[!] Failed to fetch Gitea repo: {resp.status_code} {resp.text}")
        return ""
    return resp.json().get("description") or ""


def update_codeberg_description(service_name, repo, token, base_api_url, description):
    if "[Mirror]" not in description:
        description = f"[Mirror] {description}"

    url = f"{base_api_url}/repos/{EXTERNAL_USER}/{repo}"
    headers = {"Authorization": f"token {token}"}
    payload = {"description": description.strip()}

    resp = requests.patch(url, json=payload, headers=headers)
    if resp.status_code == 200:
        print(f"[+] {service_name} description updated")
    else:
        print(f"[!] {service_name} update failed: {resp.status_code} {resp.text}")


def main():
    if len(sys.argv) < 4:
        print("Usage: python script.py <local_owner> <repo_name> <public/private>")
        return

    local_owner = sys.argv[1]
    repo_name = sys.argv[2]
    visibility = sys.argv[3]

    services = [
        {
            "name": "Codeberg",
            "token": CODEBERG_TOKEN,
            "api_base": "https://codeberg.org/api/v1",
            "host": "codeberg.org"
        },
    ]

    for svc in services:
        desc = get_gitea_description(local_owner, repo_name)

        remote_url = f"https://{EXTERNAL_USER}:{svc['token']}@{svc['host']}/{EXTERNAL_USER}/{repo_name}.git"

        # Create repo if missing (does not block mirror addition)
        create_codeberg_repo(svc["name"], repo_name, svc["token"], svc["api_base"], visibility)

        # Always attempt to add mirror
        add_gitea_mirror(local_owner, repo_name, svc["name"], remote_url)

        # Update description regardless
        update_codeberg_description(svc["name"], repo_name, svc["token"], svc["api_base"], desc)


if __name__ == "__main__":
    main()
