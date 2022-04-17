import requests


def handler(event, context):
    response = requests.get("https://example.com")
    print(response.text)
    return "Hello World!"
