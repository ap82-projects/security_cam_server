# security_cam_server

## API Endpoints

### POST /api/user
Creates a new user.  The following format is passed to the body
\{
  "email": email address as string
  "googleid: google id as string
  "incidents": \[\]
  "name": name as string
  "phone": phone number as string
  "watching": false
\}

It returns the newly created user's document id in the following format
\{
   "id": document id as string
\}

### GET /api/user?id=<"user document id as string">
Returns specified user's data in the following format
\{
  "email": email address as string
  "googleid": google id as string
  "incidents": \[\{
                 "time": time of incident as string
                 "image": image taken as string
               \}\],
  "name": name as string
  "phone": phone number as string
  "watching": status of whether the user is watching as a boolean
\}

### DELETE /api/user?id=<"user id as string">
Deletes specified user from database

### GET /api/user/google?id=<"google id as string">
If a user with the specified google id exists, document id
is returned in the following format
\{
  "id": document id as string
\}

### PUT /api/user/incident?id=<"user id as string">
Used to add incidents to the user's data.  Takes and object in the following format
\{
  "time": time of incident as string
  "image": image taken as string
\}

### DELETE /api/user/incident?id=<"user id as string">&time=<"time of incident as string">
Deletes specified incident from users data

### /api/user/watching?id=<"user id as string">
Sets whether the user is currently trying to watch the live camera feed.  Can be set with the following.
\{
  "watching": boolean representing whether the user is watching or not
\}
