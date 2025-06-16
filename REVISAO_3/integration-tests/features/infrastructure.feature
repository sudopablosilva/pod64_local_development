Feature: Infrastructure Setup
  As a system administrator
  I want to ensure the infrastructure is properly set up
  So that the job processing pipeline can function correctly

  Scenario: LocalStack health check
    Given LocalStack is running
    When I check the LocalStack health endpoint
    Then LocalStack should respond with healthy status

  Scenario: DynamoDB tables are created
    Given LocalStack is running
    When I list the DynamoDB tables
    Then I should see the following tables:
      | table          |
      | jobs           |
      | schedules      |
      | adapters       |
      | queue_messages |

  Scenario: SQS queues are created
    Given LocalStack is running
    When I list the SQS queues
    Then I should see the following queues:
      | queue         |
      | job-requests  |
      | jmw-queue     |
      | jmr-queue     |
      | sp-queue      |
      | spa-queue     |
      | spaq-queue    |

  Scenario: SQS message sending and receiving
    Given SQS queues are available
    When I send a test message to the job-requests queue
    Then the message should be successfully queued
    And I should be able to receive the message from the queue