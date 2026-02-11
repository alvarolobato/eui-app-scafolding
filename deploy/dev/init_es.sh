#!/usr/bin/env bash
# Initialize Elasticsearch indices and create an API key for the application.
set -e

ES_URL="http://localhost:9200"
ES_USER="admin"
ES_PASSWORD="changeme"
ES_AUTH="$ES_USER:$ES_PASSWORD"

echo "Waiting for Elasticsearch to be ready..."
until curl -s -u "$ES_AUTH" "$ES_URL/_cluster/health" | grep -q '"status":"green"\|"status":"yellow"'; do
  sleep 2
done
echo "Elasticsearch is ready."

# Create the app-sessions index for user session storage
echo "Creating app-sessions index..."
curl -s -X PUT -u "$ES_AUTH" "$ES_URL/app-sessions" \
  -H "Content-Type: application/json" \
  -d '{
    "mappings": {
      "properties": {
        "user_id": { "type": "keyword" },
        "email": { "type": "keyword" },
        "name": { "type": "text" },
        "picture": { "type": "keyword" },
        "created_at": { "type": "date" },
        "updated_at": { "type": "date" },
        "google": {
          "properties": {
            "refresh_token": { "type": "keyword" },
            "issued_at": { "type": "date" }
          }
        }
      }
    }
  }' || echo "Index may already exist"

# Create the app-data index for sample application data
echo "Creating app-data index..."
curl -s -X PUT -u "$ES_AUTH" "$ES_URL/app-data" \
  -H "Content-Type: application/json" \
  -d '{
    "mappings": {
      "properties": {
        "name": { "type": "text" },
        "description": { "type": "text" },
        "created_at": { "type": "date" },
        "updated_at": { "type": "date" },
        "status": { "type": "keyword" },
        "category": { "type": "keyword" }
      }
    }
  }' || echo "Index may already exist"

# Create an API key for the application
echo "Creating API key for the application..."
API_KEY_RESPONSE=$(curl -s -X POST -u "$ES_AUTH" "$ES_URL/_security/api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "app-api-key",
    "role_descriptors": {
      "app-role": {
        "cluster": ["monitor"],
        "index": [
          {
            "names": ["app-*"],
            "privileges": ["all"]
          }
        ]
      }
    }
  }')

API_KEY=$(echo "$API_KEY_RESPONSE" | grep -o '"encoded":"[^"]*"' | cut -d'"' -f4)

if [ -n "$API_KEY" ]; then
  echo "API key created successfully."
  # Create or update the elasticsearch secret with the API key
  kubectl create secret generic elasticsearch \
    --from-literal=api_key="$API_KEY" \
    --dry-run=client -o yaml | kubectl apply -f -
  echo "Elasticsearch secret updated with API key."
else
  echo "Warning: Could not create API key. Response: $API_KEY_RESPONSE"
fi

echo "Elasticsearch initialization complete."
