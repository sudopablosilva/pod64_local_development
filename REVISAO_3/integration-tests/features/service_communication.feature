Feature: Service Communication
  As a system administrator
  I want to ensure services communicate with each other
  So that the job processing pipeline works correctly

  Background:
    Given LocalStack is running
    And SQS queues are available

  Scenario: Message flows through the entire pipeline
    Given I have a test message
    When I send the message to the job-requests queue
    Then JMI should receive and process the message
    And JMI should forward the message to JMW queue
    And JMW should receive and process the message
    And JMW should forward the message to JMR queue
    And JMR should receive and process the message
    And JMR should forward the message to SP queue
    And SP should receive and process the message
    And SP should forward the message to SPA queue
    And SPA should receive and process the message
    And SPA should forward the message to SPAQ queue
    And SPAQ should receive and process the message

  Scenario: Service health endpoints
    When I call the health endpoint of Control-M
    Then I should receive a healthy response
    When I call the health endpoint of JMI
    Then I should receive a healthy response
    When I call the health endpoint of JMW
    Then I should receive a healthy response
    When I call the health endpoint of JMR
    Then I should receive a healthy response
    When I call the health endpoint of Scheduler Plugin
    Then I should receive a healthy response
    When I call the health endpoint of SPA
    Then I should receive a healthy response
    When I call the health endpoint of SPAQ
    Then I should receive a healthy response