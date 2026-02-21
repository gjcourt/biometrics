# Vitals Database

Vitals uses a PostgreSQL database to store user data, weight records, and water intake records.

## Tables

### Users
- `id`: UUID
- `username`: String
- `password_hash`: String

### Weight
- `id`: UUID
- `user_id`: UUID (Foreign Key)
- `weight`: Float
- `date`: Timestamp

### Water
- `id`: UUID
- `user_id`: UUID (Foreign Key)
- `amount`: Float
- `date`: Timestamp
