# Deployment

If you want to deploy your own instance of traQ, then follow this section.

## Requirements

Docker is highly recommended for production usage.

- Docker
- docker-compose

## Backend Configuration

traQ uses `/app/config.yml` (by default) for configuring the backend application.

Config value precedence:
1. Values in `config.yml`
2. Environment variable prefixed by `TRAQ_` (e.g. `TRAQ_ORIGIN`)
3. Default values

Here are some tips for configuring traQ:
- If you want a public instance, set `allowSignUp` to `true`.
    - Setting up some external OAuth2 providers (`externalAuth`) may help users signup using existing accounts.
- If you want a private instance, set `allowSignUp` to `false`.
    - You can use `externalAuth.github.allowedOrganizations` to only allow signup of your GitHub organization members.
    - Otherwise, an admin or external app has to manually set accounts up via `POST /api/v3/users`.
- For the maximum user experience, try to configure Elasticsearch, FCM, and Skyway to enable message search, notification, and Qall features, respectively.

The following are example configurations.

<details>

<summary>Example configuration</summary>

```yaml
# Server origin.
origin: https://example.com
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

# (optional) Image resizing settings.
imaging:
  # (optional) Maximum number of pixels before resizing.
  # Higher number means more memory requirement.
  maxPixels: 4096000 # 2560x1600
  # (optional) Maximum imaging concurrency.
  # Higher number means more CPU / memory requirement.
  concurrency: 1

# MariaDB settings.
# Use MariaDB 10.6.4 for maximum compatibility.
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
  username: elastic
  password: password

# Storage settings for uploaded files.
storage:
  # Storage type.
  #   local: Local storage. (default)
  #   swift: Swift object storage.
  #   s3: Amazon S3 object storage.
  #   composite: Local and Swift object storage.
  #              User icons, stamps, and thumbnails are stored locally,
  #              other uploaded files are stored in Swift object storage.
  #   memory: Store all files on memory (don't use this in production!).
  type: composite
  
  # Set this if type is "local" or "composite"
  local:
    dir: /app/storage
  
  # Set this if type is "swift" or "composite" and "composite.remote = swift"
  swift:
    username: username # Username
    apiKey: apiKey # Key for API access
    tenantName: tenantName # Tenant name
    tenantId: tenantId # Tenant ID
    container: container # Container name
    authUrl: authUrl # Authentication URL
    tempUrlKey: tempUrlKey # (optional) Secret key to issue temporary URL for objects
    cacheDir: /app/storagecache # Local directory to cache user icons, stamps, and thumbnails
  
  # Set this if type is "s3" or "composite" and "composite.remote = s3"
  s3:
    bucket: bucket # Bucket name
    region: region # Region
    endpoint: endpoint # (optional) Endpoint URL
    accessKey: accessKey # Access key
    secretKey: secretKey # Secret key
    cacheDir: /app/storagecache # Local directory to cache user icons, stamps, and thumbnails
  
  # Set this if type is "composite"
  composite:
    remote: s3

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
# Configure one or more of the following OAuth2 providers to allow signup and/or login via external accounts.
#
# Set http(s)://{{ origin }}/api/auth/{{ extAuthName }}/callback to callback URL.
# e.g. https://example.com/api/auth/github/callback for GitHub OAuth2 app.
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
  slack:
    clientId: clientId
    clientSecret: clientSecret
    allowSignUp: true
    allowedTeamId: teamId
```

</details>

Minimal configuration (with ES, no FCM, and no Skyway)

```yaml
origin: https://example.com
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
  username: elastic
  password: password

storage:
  type: local
  local:
    dir: /app/storage
```

## Frontend Configuration

Once you have configured the backend, next you need to configure the frontend.

traQ uses `config.js` for configuring the frontend application.

<details>

<summary>Example configuration</summary>

```js
;(() => {
  const config = {
    // (optional) Application name
    // You must set the same value to APP_NAME env.
    name: 'traQ',
    // (optional) Firebase Cloud Messaging (FCM) settings.
    firebase: {
      apiKey: 'apiKey',
      appId: 'appId',
      projectId: 'projectId',
      messagingSenderId: 'messagingSenderId'
    },
    // (optional) Skyway settings.
    skyway: {
      apiKey: 'apiKey'
    },
    // (optional) Enable search feature.
    enableSearch: true,
    // (optional) Application links.
    services: [
      {
        label: 'Wiki',
        iconPath: 'wiki.svg',
        appLink: 'https://wiki.example.com'
      }
    ],
    // (optional) OGP of any pages of these hosts will not be shown.
    ogpIgnoreHostNames: [
      'wiki.example.com'
    ],
    // (optional) Link to User page of wiki.
    wikiPageOrigin: 'https://wiki.example.com',
    // Show root channel create button.
    isRootChannelSelectableAsParentChannel: true,
    // (optional) Message shown when a large file was tried to post.
    tooLargeFileMessage: '大きい%sの共有にはGoogleDriveを使用してください',
    // (optional) Show copy widget link button.
    showWidgetCopyButton: true,
    // (optional) Disable inline reply feature when the message is from these channels.
    inlineReplyDisableChannels: ['#general']
  }

  self.traQConfig = config
})()
```

</details>

Minimal configuration (with ES, no FCM, and no Skyway)

```js
;(() => {
  const config = {
    enableSearch: true,
    isRootChannelSelectableAsParentChannel: true
  }

  self.traQConfig = config
})()
```

For more information, see [the type definition](https://github.com/traPtitech/traQ_S-UI/blob/master/src/types/config.d.ts).

Also you can override these files.
- [`/img/icons`](https://github.com/traPtitech/traQ_S-UI/tree/master/public/img/icons): favicon, PWA icons etc...
- [`/img/services`](https://github.com/traPtitech/traQ_S-UI/tree/master/public/img/services): Icons used for application links.

<details>

<summary>Tips for creating application link icons</summary>

- Use svg files.
  - Images other than svg are supported but not recommended.
- When using svg
  - You should use `fill: currentColor`. The theme color will be applied then.
  - Avoid `height` and `width` attributes set on the root element (`<svg>`).
- Set background transparent.

</details>

If you want, you can change the default theme with `defaultTheme.js` ([Default file](https://github.com/traPtitech/traQ_S-UI/blob/master/public/defaultTheme.js)).  
By using `THEME_COLOR` env, you can set `<meta name="theme-color">` and `<meta name="msapplication-TileColor">` value.

## Connecting the Components

Configure the rest of the required components, and connect them in `docker-compose`.

- [Reverse proxy (Caddy)](https://hub.docker.com/_/caddy) which will accept HTTP(S) requests
- traQ backend
- [traQ frontend](https://github.com/traPtitech/traQ_S-UI)
- [traQ Widget](https://github.com/traPtitech/traQ-Widget)
- [MariaDB](https://hub.docker.com/_/mariadb)
- (optional) [Elasticsearch with Sudachi plugin](https://github.com/orgs/traPtitech/packages/container/package/es-with-sudachi) (Sudachi is a Japanese analyzer)

Below is an example `docker-compose.yaml` file, configured to work with the above "Minimal configuration" `config.yml` and "Minimal configuration" `config.js` (placed inside `override` directory), plus `Caddyfile` and `es_jvm.options` below.

```yaml
version: '3'

services:
  reverse-proxy:
    image: caddy:latest
    container_name: traq-reverse-proxy
    restart: always
    ports:
      - "80:80"
      - "443:443"
    depends_on:
      - backend
      - frontend
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data
      - caddy-config:/config

  backend:
    image: ghcr.io/traptitech/traq:latest
    container_name: traq-backend
    restart: always
    expose:
      - "3000"
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
    expose:
      - "80"
    environment:
      THEME_COLOR: '#0D67EA' # this is the default value
    volumes:
      - ./override/:/app/override

  widget:
    image: ghcr.io/traptitech/traq-widget:latest
    container_name: traq-widget
    restart: always
    expose:
      - "80"

  db:
    image: mariadb:10.6.4
    container_name: traq-db
    restart: always
    environment:
      MYSQL_USER: traq
      MYSQL_PASSWORD: password
      MYSQL_ROOT_PASSWORD: password
      MYSQL_DATABASE: traq
    command: mysqld --character-set-server=utf8 --collation-server=utf8_general_ci
    expose:
      - "3306"
    volumes:
      - db:/var/lib/mysql

  es:
    image: ghcr.io/traptitech/es-with-sudachi:8.8.1-3.1.0
    container_name: traq-es
    restart: always
    environment:
      - discovery.type=single-node
      - ELASTIC_PASSWORD=password
    expose:
      - "9200"
    volumes:
      - ./es_jvm.options:/usr/share/elasticsearch/config/jvm.options.d/es_jvm.options
      - es:/usr/share/elasticsearch/data

volumes:
  caddy-data:
  caddy-config:
  app-storage:
  db:
  es:
```

`./Caddyfile`
```
example.com {
    handle /api/* {
        reverse_proxy backend:3000
    }
    handle /widget {
        uri strip_prefix /widget
        reverse_proxy widget:80
    }
    handle /widget/* {
        uri strip_prefix /widget
        reverse_proxy widget:80
    }
    handle {
        reverse_proxy frontend:80
    }
}
```

`./es_jvm.options`
```
-Xms512m
-Xmx512m
```

Run `docker-compose up -d`, and you're ready to go!

<!-- TODO: For more on actually operating the service, refer to [wiki](https://github.com/traPtitech/traQ/wiki). -->
