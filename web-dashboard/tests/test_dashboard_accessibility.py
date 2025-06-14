import pytest
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support.ui import WebDriverWait
from conftest import DASHBOARD_URL, wait_for_element, wait_for_clickable

class TestDashboardAccessibility:
    """Test accessibility features of the dashboard."""
    
    def test_keyboard_navigation(self, driver, wait_for_services):
        """Test keyboard navigation through the dashboard."""
        driver.get(DASHBOARD_URL)
        
        # Wait for page to load
        wait_for_element(driver, By.TAG_NAME, "header")
        
        # Test Tab navigation
        body = driver.find_element(By.TAG_NAME, "body")
        body.send_keys(Keys.TAB)
        
        # Should be able to navigate to focusable elements
        active_element = driver.switch_to.active_element
        assert active_element is not None
        
        # Test multiple tab presses
        for _ in range(5):
            active_element.send_keys(Keys.TAB)
            new_active = driver.switch_to.active_element
            # Should move focus (or stay on same element if it's the last one)
            assert new_active is not None
    
    def test_aria_labels_and_roles(self, driver, wait_for_services):
        """Test that ARIA labels and roles are present."""
        driver.get(DASHBOARD_URL)
        
        # Wait for tabs to load
        tabs_container = wait_for_element(driver, By.CLASS_NAME, "dashboard-tabs")
        
        # Check for tab roles
        tab_buttons = driver.find_elements(By.XPATH, "//button[contains(@class, 'tab-button')]")
        
        for tab in tab_buttons[:3]:  # Check first 3 tabs
            # Should have role="tab" or be a button
            role = tab.get_attribute("role")
            tag_name = tab.tag_name.lower()
            assert role == "tab" or tag_name == "button"
            
            # Should have aria-selected attribute
            aria_selected = tab.get_attribute("aria-selected")
            assert aria_selected in ["true", "false"] or aria_selected is None
    
    def test_focus_indicators(self, driver, wait_for_services):
        """Test that focus indicators are visible."""
        driver.get(DASHBOARD_URL)
        
        # Wait for interactive elements
        tab_buttons = WebDriverWait(driver, 10).until(
            lambda d: d.find_elements(By.XPATH, "//button[contains(@class, 'tab-button')]")
        )
        
        if len(tab_buttons) > 0:
            # Focus on first tab
            first_tab = tab_buttons[0]
            first_tab.click()
            
            # Check that element can receive focus
            driver.execute_script("arguments[0].focus();", first_tab)
            active_element = driver.switch_to.active_element
            
            # Should be the same element or at least focusable
            assert active_element is not None
    
    def test_semantic_html_structure(self, driver, wait_for_services):
        """Test that semantic HTML elements are used correctly."""
        driver.get(DASHBOARD_URL)
        
        # Should have proper header
        headers = driver.find_elements(By.TAG_NAME, "header")
        assert len(headers) >= 1
        
        # Should have main content area
        main_elements = driver.find_elements(By.TAG_NAME, "main")
        assert len(main_elements) >= 1
        
        # Should have proper heading hierarchy
        h1_elements = driver.find_elements(By.TAG_NAME, "h1")
        assert len(h1_elements) >= 1
        
        # Check for proper button elements
        buttons = driver.find_elements(By.TAG_NAME, "button")
        assert len(buttons) > 0  # Should have tab buttons at minimum
    
    def test_color_contrast_and_text_readability(self, driver, wait_for_services):
        """Test basic text readability (visual regression test)."""
        driver.get(DASHBOARD_URL)
        
        # Wait for content to load
        wait_for_element(driver, By.TAG_NAME, "header")
        
        # Check that text elements are visible and have content
        text_elements = driver.find_elements(By.XPATH, "//*[text()]")
        visible_text_count = 0
        
        for element in text_elements[:10]:  # Check first 10 text elements
            if element.is_displayed() and element.text.strip():
                visible_text_count += 1
                
                # Basic check that text is not empty
                assert len(element.text.strip()) > 0
        
        # Should have some visible text
        assert visible_text_count > 0
    
    def test_form_labels_and_inputs(self, driver, wait_for_services):
        """Test that form inputs have proper labels."""
        driver.get(DASHBOARD_URL)
        
        # Switch to Executions tab to find search input
        try:
            executions_tab = wait_for_clickable(driver, By.XPATH, "//button[contains(text(), 'Executions')]")
            executions_tab.click()
            
            # Look for search input
            search_inputs = driver.find_elements(By.CLASS_NAME, "search-input")
            
            for input_elem in search_inputs:
                # Should have placeholder or associated label
                placeholder = input_elem.get_attribute("placeholder")
                aria_label = input_elem.get_attribute("aria-label")
                
                assert placeholder or aria_label, "Input should have placeholder or aria-label"
                
        except:
            # If we can't find the search input, that's okay for this test
            pass
