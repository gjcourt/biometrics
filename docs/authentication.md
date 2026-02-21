# Vitals Authentication

Vitals uses token-based authentication. Users must sign up and log in to receive an authentication token, which must be included in the headers of subsequent API requests.

## Flow
1. User signs up via `/api/auth/signup`.
2. User logs in via `/api/auth/login` and receives a token.
3. The token is stored in a cookie or local storage and sent with each request.
