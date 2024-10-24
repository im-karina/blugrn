# blugrn

Dead simple blue/green deploys. Writing this was faster than me reading docs to figure out how to do it properly with other tools. And also was more fun.

Intended usage:

`docker-compose.blu.yml`
```
services:
  blu_app:
    volumes:
      - ./log:/usr/src/app/log
      - ./data:/usr/src/app/data
    image: ghcr.io/im-karina/my-image-name:latest
    platform: linux/amd64
    restart: always
    ports:
    - "8080:3000"
```
`docker-compose.grn.yml`
```
services:
  grn_app:
    volumes:
      - ./log:/usr/src/app/log
      - ./data:/usr/src/app/data
    image: ghcr.io/im-karina/my-image-name:latest
    platform: linux/amd64
    restart: always
    ports:
    - "8081:3000"
```
`.env`
```
HTTPS_CERT_PATH=/etc/letsencrypt/live/example.com/fullchain.pem
HTTPS_KEY_PATH=/etc/letsencrypt/live/example.com/privkey.pem
SECRET_KEY=my-secret-goes-here
ENVIRONMENT=prod
# or: ENVIRONMENT=dev
```
