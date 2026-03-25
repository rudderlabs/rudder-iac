def format_user_data(properties):
    formatted = {}
    for key, value in properties.items():
        formatted_key = key.lower().replace(' ', '_')
        formatted[formatted_key] = value
    return formatted

def sanitize_email(email):
    return email.lower().strip()
