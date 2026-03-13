def format_user_data(properties):
    """Format user properties for consistency"""
    formatted = {}
    for key, value in properties.items():
        formatted_key = key.lower().replace(' ', '_')
        formatted[formatted_key] = value
    return formatted

def sanitize_email(email):
    """Sanitize email addresses"""
    return email.lower().strip()

def reverse_string(text):
    """Reverse a string"""
    return text[::-1]
