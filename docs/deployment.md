# Deployment

If you want to deploy your own instance of traQ, then follow this section.
Docker is highly recommended for production usage.

## Requirements

- Docker
- docker-compose

## Configuration

traQ uses `/app/config.yml` (by default) for configuring the application.

Config value precedence:
1. Values in `config.yml`
2. Environment variable prefixed by `TRAQ_` (e.g. `TRAQ_ORIGIN`)
3. Default values

Here are some tips for configuring traQ:
- If you want a public instance, set `allowSignUp` to `true`.
    - Setting up some external OAuth2 providers (`externalAuth`) may help users signup using existing accounts.
- If you want a private instance, set `allowSignUp` to `false`.
    - You can use `externalAuth.github.allowedOrganizations` to only allow signup of your GitHub organization members.
    - Otherwise, an admin or external app have to manually set accounts up via `POST /api/v3/users`.
- For the maximum user experience, try to configure Elasticsearch, FCM, and Skyway to enable message search, notification, and Qall feature, respectively.

The followings are example and minimal configurations.

<details>

<summary>Example configuration</summary>

```yaml
# Server origin.
origin: example.com
# Server port.
port: 3000

# (optional) Whether users are allowed to register accounts by themselves.
# Default: false
#
# If you want to either: 
# - manually register accounts
# - use external authentication
# then set this to false.
allowSignUp: true

accessLog:
  # (optional) HTTP access logs in stdout. Default: true
  enabled: true

# (optional) Imaging settings
imaging:
  # (optional) Maximum number of pixels traQ is allowed to handle.
  # Higher number means more memory requirement.
  maxPixels: 4096000 # 2560x1600
  # (optional) Maximum imaging concurrency.
  # Higher number means more CPU / memory requirement.
  concurrency: 1

# MariaDB settings.
# traQ is designed to work with ConoHa's managed DB service.
# Use MariaDB 10.0.14 for maximum compatibility.
mariadb:
  # The usual DB connection settings.
  host: db
  port: 3306
  username: traq
  password: password
  database: traq
  # (optional) Connection settings
  connection:
    # (optional) Max open connections.
    # Set 0 for unlimited connections.
    maxOpen: 20
    # (optional) Max idle connections.
    maxIdle: 2
    # (optional) Maximum amount of time a connection may be reused in seconds.
    # Set 0 for unlimited age.
    lifeTime: 0

# Elasticsearch settings.
# You must set this to enable the message search feature.
es:
  url: http://es:9200

# Storage settings for uploaded files.
storage:
  # Storage type.
  #   local: Local storage. (default)
  #   swift: Swift object storage.
  #   composite: Local and Swift object storage.
  #              User icons, stamps, and thumbnails are stored locally,
  #              other uploaded files are stored in Swift object storage.
  #   memory: Store all files on memory (don't use this in production!).
  type: composite
  
  # Set this if type is "local" or "composite"
  local:
    dir: /app/storage
  
  # Set this if type is "swift" or "composite"
  swift:
    username: username # Username
    apiKey: apiKey # Key for API access
    tenantName: tenantName # Tenant name
    tenantId: tenantId # Tenant ID
    container: container # Container name
    authUrl: authUrl # Authentication URL
    tempUrlKey: tempUrlKey # (optional) Secret key to issue temporary URL for objects
    cacheDir: /app/storagecache # Local directory to cache user icons, stamps, and thumbnails

# (optional) GCP settings.
gcp:
  serviceAccount:
    # Cloud console project ID
    projectId: my-project-id
    # Credential file
    file: /keys/gcp-service-account.json
  stackdriver:
    profiler:
      # Whether to use Stackdriver Profiler or not.
      enabled: true

# Firebase Cloud Messaging (FCM) settings.
# You must set this to enable the notification feature.
firebase:
  serviceAccount:
    # Credential file
    file: /keys/firebase-service-account.json

# (optional) OAuth2 settings.
oauth2:
  # Whether to allow refresh tokens or not. Default: false
  isRefreshEnabled: true
  # Access token expiration time in seconds. Default: 31536000 (1 year)
  accessTokenExp: 31536000 # 1 year

# Skyway settings.
# You must set this to enable the call ('Qall') feature.
skyway:
  # Skyway secret key.
  secretKey: secretKey

# (optional) JWT settings.
# Used to issue QR codes to authenticate user.
jwt:
  keys:
    private: /keys/jwt.pem

# External authentication settings.
# Set one or more of the following OAuth2 providers to allow signup and/or login via external accounts.
externalAuth:
  github:
    clientId: clientId
    clientSecret: clientSecret
    allowSignUp: true
    # (optional) Require user to be a member of at least one of the following organizations.
    allowedOrganizations:
      - traPtitech
  google:
    clientId: clientId
    clientSecret: clientSecret
    allowSignUp: true
  traq:
    origin: origin # Origin of the other traQ instance
    clientId: clientId
    clientSecret: clientSecret
    allowSignUp: true
  oidc:
    issuer: issuer
    clientId: clientId
    clientSecret: clientSecret
    allowSignUp: true
    scopes:
      - scope
```

</details>

Minimal configuration

```yaml
origin: example.com
port: 3000
allowSignUp: true

mariadb:
  host: db
  port: 3306
  username: traq
  password: password
  database: traq

es:
  url: http://es:9200

storage:
  type: local
  local:
    dir: /app/storage

firebase:
  serviceAccount:
    file: /keys/firebase-service-account.json

skyway:
  secretKey: secretKey
```

## Building traQ_S-UI (optional)

Once you have configured the traQ itself, build the [traQ_S-UI](https://github.com/traPtitech/traQ_S-UI) according to your needs.

If you have configured at least one of FCM and Skyway, you will need to build the UI image yourself:

1. Clone [traQ_S-UI](https://github.com/traPtitech/traQ_S-UI).
2. Edit [src/config.ts](https://github.com/traPtitech/traQ_S-UI/blob/master/src/config.ts).
3. Build the image: `docker build -t ghcr.io/traptitech/traq-ui:latest .`

## Server settings

Configure the rest of required components.

The following is an example `docker-compose.yaml` file.

```yaml
version: '3'

services:
  reverse-proxy:
    image: caddy:latest
    container_name: traq-reverse-proxy
    restart: always
    ports:
      - "80:80"
    depends_on:
      - backend
      - frontend
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro

  backend:
    image: ghcr.io/traptitech/traq:latest
    container_name: traq-backend
    restart: always
    ports:
      - "3000:3000"
    depends_on:
      - db
      - es
    volumes:
      - ./config.yml:/app/config.yml
      - app-storage:/app/storage

  frontend:
    image: ghcr.io/traptitech/traq-ui:latest
    container_name: traq-frontend
    restart: always
    ports:
      - "8000:80"

  widget:
    image: ghcr.io/traptitech/traq-widget:latest
    container_name: traq-widget
    restart: always
    ports:
      - "8001:80"

  db:
    image: mariadb:10.0.19
    container_name: traq-db
    restart: always
    environment:
      MYSQL_USER: traq
      MYSQL_PASSWORD: password
      MYSQL_DATABASE: traq
    command: mysqld --character-set-server=utf8 --collation-server=utf8_general_ci
    expose:
      - "3306"
    volumes:
      - db:/var/lib/mysql

  es:
    image: ghcr.io/traptitech/es-with-sudachi:7.10.2-2.1.1-SNAPSHOT
    restart: always
    environment:
      - discovery.type=single-node
    ports:
      - "9200:9200"
    volumes:
      - ./es_jvm.options:/usr/share/elasticsearch/config/jvm.options.d/es_jvm.options
      - es:/usr/share/elasticsearch/data

volumes:
  app-storage:
  db:
  es:
```

`./Caddyfile`
```
example.com {
    handle /api/* {
        reverse_proxy localhost:3000
    }
    handle /widget {
        uri strip_prefix /widget
        reverse_proxy localhost:8001
    }
    handle /widget/* {
        uri strip_prefix /widget
        reverse_proxy localhost:8001
    }
    handle {
        reverse_proxy localhost:8000
    }
}
```

`./es_jvm.options`
```
-Xms512m
-Xmx512m
```

Run `docker-compose up -d`, and you're ready to go!
