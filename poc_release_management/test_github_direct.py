#!/usr/bin/env python3
"""
Test GitHub API access WITHOUT proxy settings
"""
import os
import requests
import urllib3

# Disable SSL warnings
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

GITHUB_TOKEN = os.environ.get("GITHUB_TOKEN")
TEST_REPOS = [
    "itau-ns7-container-job-manager-worker",
    "itau-ns7-container-job-manager-runner", 
    "itau-ns7-container-scheduler-manager",
    "itau-ns7-container-scheduler-adapter"
]

def test_direct():
    print("=== TESTING DIRECT CONNECTION (NO PROXY) ===")
    
    if not GITHUB_TOKEN:
        print("[ERROR] GITHUB_TOKEN not set")
        return
    
    headers = {"Authorization": f"token {GITHUB_TOKEN}"}
    
    # Test user authentication
    print("\n1. Testing user authentication...")
    try:
        response = requests.get(
            "https://api.github.com/user", 
            headers=headers, 
            verify=False, 
            timeout=10
        )
        print(f"   Status: {response.status_code}")
        if response.ok:
            user = response.json()
            print(f"   User: {user.get('login', 'N/A')}")
        else:
            print(f"   Error: {response.text[:100]}")
    except Exception as e:
        print(f"   Exception: {type(e).__name__}: {str(e)}")
    
    # Test organization access
    print("\n2. Testing organization access...")
    try:
        response = requests.get(
            "https://api.github.com/orgs/itau-corp", 
            headers=headers, 
            verify=False, 
            timeout=10
        )
        print(f"   Status: {response.status_code}")
        if response.ok:
            org = response.json()
            print(f"   Org: {org.get('login', 'N/A')}")
        else:
            print(f"   Error: {response.text[:100]}")
    except Exception as e:
        print(f"   Exception: {type(e).__name__}: {str(e)}")
    
    # Test specific repositories
    print("\n3. Testing specific repositories...")
    for repo_name in TEST_REPOS:
        try:
            url = f"https://api.github.com/repos/itau-corp/{repo_name}"
            response = requests.get(url, headers=headers, verify=False, timeout=10)
            print(f"   {repo_name}: {response.status_code}")
            if not response.ok:
                print(f"      Error: {response.text[:100]}")
        except Exception as e:
            print(f"   {repo_name}: Exception - {type(e).__name__}: {str(e)}")

if __name__ == "__main__":
    test_direct()
