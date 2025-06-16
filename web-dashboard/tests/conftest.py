import pytest
import time
import requests
from selenium import webdriver
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from webdriver_manager.chrome import ChromeDriverManager

# Test configuration
DASHBOARD_URL = "http://localhost:3000"
API_BASE_URL = "http://localhost:4333"
TIMEOUT = 10

@pytest.fixture(scope="session")
def driver():
    """Create a Chrome WebDriver instance for testing."""
    chrome_options = Options()
    chrome_options.add_argument("--headless")  # Run in headless mode
    chrome_options.add_argument("--no-sandbox")
    chrome_options.add_argument("--disable-dev-shm-usage")
    chrome_options.add_argument("--disable-gpu")
    chrome_options.add_argument("--window-size=1920,1080")
    
    service = Service(ChromeDriverManager().install())
    driver = webdriver.Chrome(service=service, options=chrome_options)
    driver.implicitly_wait(TIMEOUT)
    
    yield driver
    
    driver.quit()

@pytest.fixture(scope="session")
def wait_for_services():
    """Wait for all services to be ready before running tests."""
    services = [
        ("Dashboard", DASHBOARD_URL + "/health"),
        ("JMI", API_BASE_URL + "/health"),
    ]
    
    max_retries = 30
    for service_name, url in services:
        for attempt in range(max_retries):
            try:
                response = requests.get(url, timeout=5)
                if response.status_code == 200:
                    print(f"✅ {service_name} is ready")
                    break
            except requests.exceptions.RequestException:
                pass
            
            if attempt == max_retries - 1:
                pytest.fail(f"❌ {service_name} failed to start after {max_retries} attempts")
            
            time.sleep(2)

@pytest.fixture
def create_test_execution():
    """Create a test execution for testing purposes."""
    execution_name = f"SELENIUM_TEST_{int(time.time())}"
    
    try:
        response = requests.post(
            f"{API_BASE_URL}/startExecution",
            json={"executionName": execution_name},
            timeout=10
        )
        if response.status_code == 200:
            data = response.json()
            return {
                "name": execution_name,
                "uuid": data.get("executionUuid"),
                "status": data.get("status")
            }
    except requests.exceptions.RequestException as e:
        pytest.fail(f"Failed to create test execution: {e}")
    
    return None

def wait_for_element(driver, by, value, timeout=TIMEOUT):
    """Wait for an element to be present and visible."""
    return WebDriverWait(driver, timeout).until(
        EC.presence_of_element_located((by, value))
    )

def wait_for_clickable(driver, by, value, timeout=TIMEOUT):
    """Wait for an element to be clickable."""
    return WebDriverWait(driver, timeout).until(
        EC.element_to_be_clickable((by, value))
    )
