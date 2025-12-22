#!/bin/bash

# Test script for sora-g channel API

URL="http://localhost:3001/relay/task"

# Test data for sora-g video generation
TEST_DATA='{
  "platform": "57",
  "action": "video.generate",
  "model": "sora-g",
  "input": {
    "prompt": "A beautiful sunset over the mountains",
    "duration": "30"
  }
}'

echo "Testing sora-g channel API at $URL"
echo "Request data: $TEST_DATA"
echo ""

# Send POST request
curl -X POST -H "Content-Type: application/json" -d "$TEST_DATA" $URL

echo ""
echo "Test completed"