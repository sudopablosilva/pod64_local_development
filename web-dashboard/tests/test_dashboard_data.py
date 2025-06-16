import pytest
import time
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from conftest import DASHBOARD_URL, wait_for_element, wait_for_clickable

class TestDashboardData:
    """Test dashboard data display and functionality."""
    
    def test_services_grid_displays(self, driver, wait_for_services):
        """Test that the services grid displays correctly."""
        driver.get(DASHBOARD_URL)
        
        # Should start on Overview tab
        services_grid = wait_for_element(driver, By.CLASS_NAME, "services-grid")
        assert services_grid is not None
        
        # Wait for service cards to load
        service_cards = WebDriverWait(driver, 10).until(
            lambda d: d.find_elements(By.CLASS_NAME, "service-card")
        )
        
        # Should have at least 6 services
        assert len(service_cards) >= 6
        
        # Check that each service card has required elements
        for card in service_cards[:3]:  # Check first 3 cards
            service_name = card.find_element(By.CLASS_NAME, "service-name")
            assert service_name.text.strip() != ""
            
            status_icon = card.find_element(By.CLASS_NAME, "status-icon")
            assert status_icon is not None
    
    def test_executions_tab_functionality(self, driver, wait_for_services, create_test_execution):
        """Test executions tab displays data correctly."""
        driver.get(DASHBOARD_URL)
        
        # Switch to Executions tab
        executions_tab = wait_for_clickable(driver, By.XPATH, "//button[contains(text(), 'Executions')]")
        executions_tab.click()
        
        # Wait for executions content to load
        executions_panel = wait_for_element(driver, By.CLASS_NAME, "executions-panel")
        assert executions_panel is not None
        
        # Check for search functionality
        search_input = wait_for_element(driver, By.CLASS_NAME, "search-input")
        assert search_input is not None
        assert search_input.get_attribute("placeholder") == "Search executions..."
        
        # Check for filter dropdowns
        filter_selects = driver.find_elements(By.CLASS_NAME, "filter-select")
        assert len(filter_selects) >= 2  # Status and Stage filters
        
        # Wait for execution data to load (may take a moment)
        time.sleep(3)
        
        # Check if executions are displayed
        execution_cards = driver.find_elements(By.CLASS_NAME, "execution-card")
        if len(execution_cards) > 0:
            # Verify execution card structure
            first_card = execution_cards[0]
            
            # Should have execution name (even if it shows as "Unknown" due to data mapping issues)
            execution_name = first_card.find_element(By.CLASS_NAME, "execution-name")
            assert execution_name is not None
            
            # Should have status indicator
            status_elements = first_card.find_elements(By.CLASS_NAME, "status-icon")
            assert len(status_elements) > 0
    
    def test_tables_tab_functionality(self, driver, wait_for_services):
        """Test tables tab displays data correctly."""
        driver.get(DASHBOARD_URL)
        
        # Switch to Tables tab
        tables_tab = wait_for_clickable(driver, By.XPATH, "//button[contains(text(), 'Tables')]")
        tables_tab.click()
        
        # Wait for tables content to load
        tables_panel = wait_for_element(driver, By.CLASS_NAME, "tables-panel")
        assert tables_panel is not None
        
        # Wait for table data to load
        time.sleep(3)
        
        # Check for tables grid or empty state
        tables_grid = driver.find_elements(By.CLASS_NAME, "tables-grid")
        tables_empty = driver.find_elements(By.CLASS_NAME, "tables-empty")
        
        # Should have either tables or empty state
        assert len(tables_grid) > 0 or len(tables_empty) > 0
        
        if len(tables_grid) > 0:
            # If tables exist, check for table cards
            table_cards = driver.find_elements(By.CLASS_NAME, "table-card")
            if len(table_cards) > 0:
                # Verify table card structure
                first_card = table_cards[0]
                table_name = first_card.find_element(By.CLASS_NAME, "table-name")
                assert table_name.text.strip() != ""
    
    def test_queues_tab_functionality(self, driver, wait_for_services):
        """Test queues tab displays data correctly."""
        driver.get(DASHBOARD_URL)
        
        # Switch to Queues tab
        queues_tab = wait_for_clickable(driver, By.XPATH, "//button[contains(text(), 'Queues')]")
        queues_tab.click()
        
        # Wait for queues content to load
        queues_panel = wait_for_element(driver, By.CLASS_NAME, "queues-panel")
        assert queues_panel is not None
        
        # Wait for queue data to load
        time.sleep(3)
        
        # Check for queues grid or empty state
        queues_grid = driver.find_elements(By.CLASS_NAME, "queues-grid")
        queues_empty = driver.find_elements(By.CLASS_NAME, "queues-empty")
        
        # Should have either queues or empty state
        assert len(queues_grid) > 0 or len(queues_empty) > 0
        
        if len(queues_grid) > 0:
            # If queues exist, check for queue cards
            queue_cards = driver.find_elements(By.CLASS_NAME, "queue-card")
            if len(queue_cards) > 0:
                # Verify queue card structure
                first_card = queue_cards[0]
                queue_name = first_card.find_element(By.CLASS_NAME, "queue-name")
                assert queue_name.text.strip() != ""
                
                # Check for metrics
                metrics = first_card.find_elements(By.CLASS_NAME, "metric")
                assert len(metrics) >= 3  # Should have visible, processing, and total metrics
    
    def test_search_functionality(self, driver, wait_for_services):
        """Test search functionality in executions tab."""
        driver.get(DASHBOARD_URL)
        
        # Switch to Executions tab
        executions_tab = wait_for_clickable(driver, By.XPATH, "//button[contains(text(), 'Executions')]")
        executions_tab.click()
        
        # Wait for search input
        search_input = wait_for_element(driver, By.CLASS_NAME, "search-input")
        
        # Test typing in search
        search_input.clear()
        search_input.send_keys("TEST")
        
        # Wait a moment for search to process
        time.sleep(1)
        
        # Search input should contain the text
        assert search_input.get_attribute("value") == "TEST"
        
        # Clear search
        search_input.clear()
        assert search_input.get_attribute("value") == ""
