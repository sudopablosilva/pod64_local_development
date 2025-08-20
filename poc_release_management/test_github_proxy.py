#!/usr/bin/env python3
"""
Test GitHub API access WITH proxy settings
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

# Common corporate proxy settings
PROXY_CONFIGS = [
    # Auto-detect from environment
    {
        "name": "Environment Variables",
        "proxies": {
            "http": os.environ.get("HTTP_PROXY") or os.environ.get("http_proxy"),
            "https": os.environ.get("HTTPS_PROXY") or os.environ.get("https_proxy")
        }
    }
]

def test_with_proxy():
    print("=== TESTING WITH PROXY CONFIGURATIONS ===")
    
    if not GITHUB_TOKEN:
        print("[ERROR] GITHUB_TOKEN not set")
        return
    
    headers = {"Authorization": f"token {GITHUB_TOKEN}"}
    
    for config in PROXY_CONFIGS:
        print(f"\n--- Testing {config['name']} ---")
        
        # Skip if no proxy configured
        if not any(config['proxies'].values()):
            print("   No proxy configured, skipping...")
            continue
            
        print(f"   HTTP Proxy: {config['proxies'].get('http', 'None')}")
        print(f"   HTTPS Proxy: {config['proxies'].get('https', 'None')}")
        
        # Test user authentication
        print("   Testing user authentication...")
        try:
            response = requests.get(
                "https://api.github.com/user", 
                headers=headers, 
                proxies=config['proxies'],
                verify=False, 
                timeout=10
            )
            print(f"      Status: {response.status_code}")
            if response.ok:
                user = response.json()
                print(f"      User: {user.get('login', 'N/A')}")
            else:
                print(f"      Error: {response.text[:100]}")
        except Exception as e:
            print(f"      Exception: {type(e).__name__}: {str(e)}")
        
        # Test one repository
        print("   Testing repository access...")
        try:
            url = f"https://api.github.com/repos/itau-corp/{TEST_REPOS[0]}"
            response = requests.get(
                url, 
                headers=headers, 
                proxies=config['proxies'],
                verify=False, 
                timeout=10
            )
            print(f"      {TEST_REPOS[0]}: {response.status_code}")
            if not response.ok:
                print(f"         Error: {response.text[:100]}")
        except Exception as e:
            print(f"      {TEST_REPOS[0]}: Exception - {type(e).__name__}: {str(e)}")

def show_environment():
    print("=== ENVIRONMENT VARIABLES ===")
    proxy_vars = ["HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy", "NO_PROXY", "no_proxy"]
    for var in proxy_vars:
        value = os.environ.get(var)
        print(f"{var}: {value if value else 'Not set'}")

if __name__ == "__main__":
    show_environment()
    test_with_proxy()
