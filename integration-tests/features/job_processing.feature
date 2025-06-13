Feature: Job Processing Pipeline
  As a system administrator
  I want to submit jobs through Control-M
  So that they are processed through the entire pipeline

  Background:
    Given all services are running
    And LocalStack is initialized with required resources

  Scenario: Submit a simple job and process it through the pipeline
    Given I have a job request with the following details:
      | field       | value           |
      | job_name    | test-job-001    |
      | job_type    | shell           |
      | priority    | 1               |
    When I submit the job to Control-M
    Then the job should be accepted with status "submitted"
    And the job should appear in JMI with status "integrated"
    And the job should be processed by JMW with status "processed"
    And the job should be executed by JMR with status "executed"
    And a schedule should be created by Scheduler Plugin
    And an adapter should be configured by SPA
    And a queue message should be created by SPAQ

  Scenario: Submit multiple jobs with different priorities
    Given I have multiple job requests:
      | job_name     | job_type | priority |
      | high-job     | python   | 1        |
      | medium-job   | sql      | 2        |
      | low-job      | shell    | 3        |
    When I submit all jobs to Control-M
    Then all jobs should be processed through the pipeline
    And jobs should be processed according to their priority

  Scenario: Verify service health endpoints
    When I check the health of all services
    Then all services should respond with healthy status

  Scenario: Verify data persistence
    Given I submit a job through the pipeline
    When I query the DynamoDB tables
    Then the job data should be persisted correctly
    And the schedule data should be stored
    And the adapter configuration should be saved
    And the queue messages should be recorded
