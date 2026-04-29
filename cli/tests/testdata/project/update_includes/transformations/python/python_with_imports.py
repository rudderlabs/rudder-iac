import json
import hashlib
from datetime import datetime
import base64

from pythonUtilsLibrary import format_user_data, sanitize_email

def transformEvent(event, metadata):
    event['timestamp'] = datetime.now().isoformat()

    if 'userId' in event:
        user_hash = hashlib.sha256(event['userId'].encode()).hexdigest()
        event['userHash'] = user_hash[:8]

    if 'properties' in event:
        event['properties'] = format_user_data(event['properties'])

    if 'email' in event:
        event['email'] = sanitize_email(event['email'])

    if 'data' in event:
        encoded = base64.b64encode(json.dumps(event['data']).encode()).decode()
        event['encodedData'] = encoded

    return event
