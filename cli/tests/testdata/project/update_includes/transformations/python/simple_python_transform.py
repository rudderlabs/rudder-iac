def transformEvent(event, metadata):
    if 'properties' not in event:
        event['properties'] = {}

    event['properties']['message'] = 'Hello from Python!'
    event['properties']['version'] = '2.0'

    if 'userId' in event:
        event['properties']['formatted_user'] = f"User: {event['userId']}"

    return event
