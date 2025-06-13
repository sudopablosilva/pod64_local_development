#!/bin/bash

# Fix AWS SDK v2 type issues in all services
for service in jmi jmw jmr scheduler-plugin spa spaq; do
    echo "Fixing AWS types in $service..."
    cd $service
    
    # Add types import
    sed -i '' 's|"github.com/aws/aws-sdk-go-v2/service/sqs"|"github.com/aws/aws-sdk-go-v2/service/sqs"\n\t"github.com/aws/aws-sdk-go-v2/service/sqs/types"|g' main.go
    
    # Fix Message type
    sed -i '' 's|sqs\.Message|types.Message|g' main.go
    
    # Fix AttributeValue type  
    sed -i '' 's|dynamodb\.AttributeValue|types.AttributeValue|g' main.go
    
    cd ..
done
