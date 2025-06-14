import pytest
import time
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from conftest import DASHBOARD_URL, wait_for_element, wait_for_clickable

class TestDashboardBasic:
    """Basic dashboard functionality tests."""
    
    def test_dashboard_loads(self, driver, wait_for_services):
        """Test that the dashboard loads successfully."""
        driver.get(DASHBOARD_URL)
        
        # Wait for the header to load
        header = wait_for_element(driver, By.TAG_NAME, "header")
        assert header is not None
        
        # Check for the main title
        title = wait_for_element(driver, By.XPATH, "//h1[contains(text(), 'POC BDD Dashboard')]")
        assert title is not None
        assert "POC BDD Dashboard" in title.text
    
    def test_connection_status_indicator(self, driver, wait_for_services):
        """Test that connection status is displayed."""
        driver.get(DASHBOARD_URL)
        
        # Wait for connection status to appear
        connection_status = wait_for_element(driver, By.CLASS_NAME, "connection-status")
        assert connection_status is not None
        
        # Should show connected status
        WebDriverWait(driver, 10).until(
            lambda d: "connected" in connection_status.get_attribute("class").lower() or
                     "connecting" in connection_status.get_attribute("class").lower()
        )
    
    def test_navigation_tabs(self, driver, wait_for_services):
        """Test that all navigation tabs are present and clickable."""
        driver.get(DASHBOARD_URL)
        
        # Wait for tabs to load
        tabs_container = wait_for_element(driver, By.CLASS_NAME, "dashboard-tabs")
        assert tabs_container is not None
        
        # Check for all expected tabs
        expected_tabs = ["Overview", "Executions", "Tables", "Queues"]
        
        for tab_name in expected_tabs:
            tab = wait_for_element(driver, By.XPATH, f"//button[contains(text(), '{tab_name}')]")
            assert tab is not None
            assert tab.is_enabled()
    
    def test_tab_switching(self, driver, wait_for_services):
        """Test that tab switching works correctly."""
        driver.get(DASHBOARD_URL)
        
        # Wait for tabs to load
        tabs_container = wait_for_element(driver, By.CLASS_NAME, "dashboard-tabs")
        
        # Test switching to Executions tab
        executions_tab = wait_for_clickable(driver, By.XPATH, "//button[contains(text(), 'Executions')]")
        executions_tab.click()
        
        # Verify the tab is active
        WebDriverWait(driver, 5).until(
            lambda d: "active" in executions_tab.get_attribute("class")
        )
        
        # Test switching to Tables tab
        tables_tab = wait_for_clickable(driver, By.XPATH, "//button[contains(text(), 'Tables')]")
        tables_tab.click()
        
        # Verify the tab is active
        WebDriverWait(driver, 5).until(
            lambda d: "active" in tables_tab.get_attribute("class")
        )
    
    def test_responsive_design(self, driver, wait_for_services):
        """Test responsive design at different screen sizes."""
        driver.get(DASHBOARD_URL)
        
        # Test desktop size
        driver.set_window_size(1920, 1080)
        header = wait_for_element(driver, By.TAG_NAME, "header")
        assert header.is_displayed()
        
        # Test tablet size
        driver.set_window_size(768, 1024)
        time.sleep(1)  # Allow layout to adjust
        header = driver.find_element(By.TAG_NAME, "header")
        assert header.is_displayed()
        
        # Test mobile size
        driver.set_window_size(375, 667)
        time.sleep(1)  # Allow layout to adjust
        header = driver.find_element(By.TAG_NAME, "header")
        assert header.is_displayed()
        
        # Reset to desktop size
        driver.set_window_size(1920, 1080)
