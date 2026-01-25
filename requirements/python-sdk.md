# Python SDK Requirements

## Overview

The Python SDK provides a Pythonic interface for integrating BunBase services into Python applications, web frameworks, and data science workflows.

## Supported Python Versions

- Python 3.10+
- Python 3.11+
- Python 3.12+

## Installation

```bash
# pip
pip install bunbase

# poetry
poetry add bunbase

# pipenv
pipenv install bunbase
```

## Core Features

### 1. Initialization

```python
from bunbase import Client

# Initialize client
bunbase = Client(
    api_key='bunbase_pk_live_xxx',
    project='my-project',
    region='us-east-1',
    options={
        'timeout': 30,
        'retries': 3,
        'verify_ssl': True
    }
)

# Async client
from bunbase import AsyncClient

bunbase = AsyncClient(api_key='xxx')
```

### 2. Authentication Module

```python
# Register user
user, session = bunbase.auth.sign_up(
    email='user@example.com',
    password='securepassword',
    metadata={'name': 'John Doe'}
)

# Login
user, session = bunbase.auth.sign_in(
    email='user@example.com',
    password='securepassword'
)

# OAuth
auth_url = bunbase.auth.sign_in_with_oauth(
    provider='google',
    redirect_to='https://myapp.com/callback'
)

# Get current user
user = bunbase.auth.get_user()

# Update user
user = bunbase.auth.update_user(
    data={'name': 'Jane Doe'}
)

# Sign out
bunbase.auth.sign_out()

# Auth state listener
@bunbase.auth.on_auth_state_change
def handle_auth_change(event, session):
    print(f'Auth event: {event}')
```

### 3. Database Module

```python
# Query documents
response = bunbase.table('users') \
    .select('*') \
    .eq('status', 'active') \
    .order('created_at', desc=True) \
    .limit(10) \
    .execute()

users = response.data

# Get single document
user = bunbase.table('users') \
    .select('*') \
    .eq('id', 'user-123') \
    .single() \
    .execute() \
    .data

# Insert document
response = bunbase.table('users').insert({
    'name': 'John Doe',
    'email': 'john@example.com',
    'age': 25
}).execute()

# Batch insert
response = bunbase.table('users').insert([
    {'name': 'User 1', 'email': 'user1@example.com'},
    {'name': 'User 2', 'email': 'user2@example.com'}
]).execute()

# Update document
response = bunbase.table('users') \
    .update({'age': 26}) \
    .eq('id', 'user-123') \
    .execute()

# Delete document
response = bunbase.table('users') \
    .delete() \
    .eq('id', 'user-123') \
    .execute()

# Advanced filtering
from bunbase.filters import or_, and_

response = bunbase.table('orders') \
    .select('*, customer:customers(*), items:order_items(*)') \
    .gte('total', 100) \
    .lte('total', 1000) \
    .filter(or_(
        eq('priority', 'high'),
        gte('amount', 500)
    )) \
    .execute()

# Full-text search
response = bunbase.table('articles') \
    .select('*') \
    .text_search('title', 'machine learning') \
    .execute()

# Real-time subscriptions
def handle_insert(payload):
    print(f'New message: {payload["new"]}')

subscription = bunbase.table('messages') \
    .on('INSERT', handle_insert) \
    .subscribe()

# Unsubscribe
subscription.unsubscribe()
```

### 4. Storage Module

```python
# Upload file
with open('avatar.jpg', 'rb') as f:
    response = bunbase.storage \
        .from_('avatars') \
        .upload('public/user-123.jpg', f, {
            'cache_control': '3600',
            'upsert': True
        })

# Download file
response = bunbase.storage \
    .from_('avatars') \
    .download('public/user-123.jpg')

with open('downloaded.jpg', 'wb') as f:
    f.write(response)

# List files
files = bunbase.storage \
    .from_('avatars') \
    .list('public/', {
        'limit': 100,
        'sort_by': {'column': 'created_at', 'order': 'desc'}
    })

# Delete file
bunbase.storage \
    .from_('avatars') \
    .remove(['public/user-123.jpg'])

# Get public URL
url = bunbase.storage \
    .from_('avatars') \
    .get_public_url('public/user-123.jpg', {
        'transform': {
            'width': 200,
            'height': 200,
            'format': 'webp'
        }
    })

# Create signed URL
signed_url = bunbase.storage \
    .from_('private-files') \
    .create_signed_url('documents/contract.pdf', 3600)

# Upload with progress
def progress_callback(progress):
    print(f'Upload progress: {progress}%')

with open('large-file.mp4', 'rb') as f:
    bunbase.storage \
        .from_('videos') \
        .upload('large-video.mp4', f, {
            'on_progress': progress_callback
        })
```

### 5. Functions Module

```python
# Invoke function
response = bunbase.functions.invoke('send-email', {
    'to': 'user@example.com',
    'subject': 'Welcome!',
    'template': 'welcome'
})

# Invoke with custom headers
response = bunbase.functions.invoke('protected-function',
    body={'action': 'delete'},
    headers={'X-Custom-Header': 'value'}
)

# Async invocation
import asyncio

async def call_function():
    response = await bunbase.functions.invoke_async('generate-report', {
        'report_type': 'monthly'
    })
    return response

result = asyncio.run(call_function())
```

### 6. Real-time Module

```python
# Connect to real-time
realtime = bunbase.realtime

# Subscribe to channel
channel = realtime.channel('chat:room-123')

@channel.on_subscribe
def on_subscribed(status):
    print(f'Subscription status: {status}')

channel.subscribe()

# Send messages
channel.send({
    'type': 'broadcast',
    'event': 'new-message',
    'payload': {
        'text': 'Hello!',
        'user_id': 'user-123'
    }
})

# Listen to messages
@channel.on('new-message')
def handle_message(payload):
    print(f'New message: {payload}')

# Presence
presence_channel = realtime.channel('presence:room-123', {
    'presence': True
})

presence_channel.subscribe()

# Track presence
presence_channel.track({
    'user': 'John',
    'status': 'online'
})

# Listen to presence
@presence_channel.on_presence('join')
def on_user_join(payload):
    print(f'User joined: {payload}')

@presence_channel.on_presence('leave')
def on_user_leave(payload):
    print(f'User left: {payload}')
```

## Async/Await Support

```python
import asyncio
from bunbase import AsyncClient

async def main():
    bunbase = AsyncClient(api_key='xxx')

    # Async queries
    response = await bunbase.table('users') \
        .select('*') \
        .execute()

    # Async file upload
    with open('file.jpg', 'rb') as f:
        await bunbase.storage \
            .from_('uploads') \
            .upload('file.jpg', f)

    # Async function invocation
    result = await bunbase.functions.invoke('my-function', {
        'data': 'value'
    })

asyncio.run(main())
```

## Type Hints & IDE Support

```python
from typing import List, Dict, Optional
from bunbase import Client
from bunbase.types import User, DatabaseResponse

bunbase: Client = Client(api_key='xxx')

# Type-hinted queries
response: DatabaseResponse[User] = bunbase.table('users') \
    .select('*') \
    .execute()

users: List[User] = response.data

# Optional types
user: Optional[User] = bunbase.table('users') \
    .eq('id', 'user-123') \
    .single() \
    .execute() \
    .data
```

## Framework Integrations

### Django Integration

```python
# settings.py
BUNBASE = {
    'API_KEY': 'bunbase_pk_live_xxx',
    'PROJECT': 'my-project',
}

# views.py
from bunbase.django import get_client

def my_view(request):
    bunbase = get_client()
    users = bunbase.table('users').select('*').execute().data
    return render(request, 'template.html', {'users': users})

# Authentication backend
AUTHENTICATION_BACKENDS = [
    'bunbase.django.auth.BunBaseBackend',
]
```

### Flask Integration

```python
from flask import Flask
from bunbase.flask import BunBase

app = Flask(__name__)
app.config['BUNBASE_API_KEY'] = 'bunbase_pk_live_xxx'

bunbase = BunBase(app)

@app.route('/users')
def get_users():
    users = bunbase.table('users').select('*').execute().data
    return {'users': users}
```

### FastAPI Integration

```python
from fastapi import FastAPI, Depends
from bunbase.fastapi import get_client, require_auth

app = FastAPI()

@app.get('/users')
async def get_users(bunbase=Depends(get_client)):
    response = await bunbase.table('users').select('*').execute()
    return response.data

@app.get('/protected')
async def protected_route(user=Depends(require_auth)):
    return {'user': user}
```

## Data Science Integration

### Pandas Integration

```python
import pandas as pd
from bunbase import Client

bunbase = Client(api_key='xxx')

# Query to DataFrame
response = bunbase.table('orders').select('*').execute()
df = pd.DataFrame(response.data)

# DataFrame to BunBase
df.to_bunbase(bunbase, 'orders', if_exists='append')

# Batch operations
for chunk in pd.read_csv('large_file.csv', chunksize=1000):
    bunbase.table('data').insert(chunk.to_dict('records')).execute()
```

### NumPy Integration

```python
import numpy as np

# Numerical data handling
data = bunbase.table('measurements').select('*').execute().data
values = np.array([d['value'] for d in data])

stats = {
    'mean': float(np.mean(values)),
    'std': float(np.std(values)),
    'min': float(np.min(values)),
    'max': float(np.max(values))
}
```

## Error Handling

```python
from bunbase.exceptions import (
    BunBaseException,
    AuthException,
    DatabaseException,
    StorageException,
    RateLimitException
)

try:
    response = bunbase.table('users').select('*').execute()
    if response.error:
        raise response.error
except AuthException as e:
    print(f'Authentication error: {e.message}')
except DatabaseException as e:
    print(f'Database error: {e.code} - {e.message}')
except RateLimitException as e:
    print(f'Rate limited. Retry after: {e.retry_after}s')
except BunBaseException as e:
    print(f'BunBase error: {e}')
```

## Connection Pooling

```python
from bunbase import Client

bunbase = Client(
    api_key='xxx',
    options={
        'pool_size': 10,
        'pool_maxsize': 20,
        'pool_timeout': 30,
        'keepalive': True
    }
)
```

## Retry & Backoff

```python
from bunbase import Client
from bunbase.retry import ExponentialBackoff

bunbase = Client(
    api_key='xxx',
    options={
        'retries': 3,
        'backoff': ExponentialBackoff(
            base=1,  # seconds
            max_delay=60,
            jitter=True
        )
    }
)
```

## Logging

```python
import logging
from bunbase import Client

# Enable debug logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger('bunbase')

bunbase = Client(
    api_key='xxx',
    options={
        'logger': logger,
        'log_level': 'DEBUG'
    }
)
```

## Testing Utilities

```python
from bunbase.testing import MockClient

def test_user_query():
    mock_bunbase = MockClient()

    # Mock response
    mock_bunbase.table('users').select.return_value.execute.return_value = {
        'data': [{'id': '1', 'name': 'Test User'}],
        'error': None
    }

    # Test your code
    response = mock_bunbase.table('users').select('*').execute()
    assert len(response['data']) == 1
```

## CLI Integration

```python
# Use in scripts
if __name__ == '__main__':
    import sys
    from bunbase import Client

    bunbase = Client(api_key=sys.argv[1])

    # Perform operations
    users = bunbase.table('users').select('*').execute().data

    for user in users:
        print(f'{user["id"]}: {user["name"]}')
```

## Performance Features

- Connection pooling
- Request batching
- Response caching
- Lazy loading
- Pagination helpers
- Bulk operations
- Compression support

## Documentation Requirements

- Getting started guide
- API reference
- Django integration guide
- Flask integration guide
- FastAPI integration guide
- Data science guide
- Migration guide
- Examples repository
