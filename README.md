# security_cam_server

## API Endpoints

### POST /api/user
Creates a new user.  The following format is passed to the body
{
  "email": email address as string
  "googleid: google id as string
  "incidents": []
  "name": name as string
  "phone": phone number as string
  "watching": false
}

It returns the newly created user's id in the following format
{ "id": id as string}

### GET /api/user?id=<"google id as string">
If a user with the specified google id exists, their data is returned in the following format
{
  "email": email address as string
  "googleid": google id as string
  "incidents": [{
                 "time": time of incident as string
                 "image": image taken as string
               }],
  "name": name as string
  "phone": phone number as string
  "watching": status of whether the user is watching as a boolean
}
### /api/user
### /api/user
### /api/user
### /api/user
### /api/user
### /api/user
### /api/user
### /api/user
### /api/user