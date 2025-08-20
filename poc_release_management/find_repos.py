#!/usr/bin/env python3
import os
import requests
import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

GITHUB_TOKEN = os.environ.get("GITHUB_TOKEN")
headers = {"Authorization": f"token {GITHUB_TOKEN}"}

print("=== FINDING REPOSITORIES IN itau-corp ===")

try:
    response = requests.get("https://api.github.com/orgs/itau-corp/repos?per_page=100", headers=headers, verify=False)
    if response.ok:
        repos = response.json()
        print(f"Found {len(repos)} repositories:")
        
        # Filter for relevant repos
        keywords = ["job-manager", "scheduler", "container", "ns7"]
        relevant = [r['name'] for r in repos if any(k in r['name'].lower() for k in keywords)]
        
        print("\nRelevant repositories:")
        for repo in sorted(relevant):
            print(f"  - {repo}")
            
        print("\nAll repositories:")
        for repo in sorted([r['name'] for r in repos]):
            print(f"  - {repo}")
    else:
        print(f"Error: {response.status_code} - {response.text}")
except Exception as e:
    print(f"Exception: {e}")
