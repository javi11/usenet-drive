# usenet-drive (WIP)

## Description

This is a simple script that allows you to mount a usenet server as a webdav drive.

**_This is not a tool to mount any nzb files, nzb files that are supported on the tool needs to be created by this tool._**

**_Use at your own risk_**

## Usage with rclone

### Install rclone

```bash
curl https://rclone.org/install.sh | sudo bash
```

### Configure rclone

```bash
rclone config
```

Add a new webdav remote with the following parameters:

- **Name**: `usenet`
- **URL**: `http://localhost:8080`

### It's hight recommended to use an rclone crypt remote to encrypt your data since this tool just obfuscate the file names.

Add a new crypt remote with the following parameters:

- **Name**: `usenet-crypt`
- **Storage provider**: `Crypt`
- **Encrypt the filenames**: `off`
- **Option directory_name_encryption**: `Don't encrypt directory names, leave them intact`

````bash

### Mount the remote

```bash
rclone mount --allow-other --async-read=true --dir-cache-time=1000h --buffer-size=32M --poll-interval=15s --rc --rc-no-auth --rc-addr=localhost:5572 --use-mmap --vfs-read-ahead=128M --vfs-read-chunk-size=32M --vfs-read-chunk-size-limit=2G --vfs-cache-max-age=504h --vfs-cache-mode=full --vfs-cache-poll-interval=30s --vfs-cache-max-size=50G --timeout=10m usenet: ${PATH_TO_MOUNT} --umask=002
````

**_When mounting it's high recommended to use a vfs cache to avoid problems with uploads and downloads._**

### API

An API to control the server, is available at `http://localhost:8081/api/v1/`.

### WebAdmin

WebAdmin, is available at `http://localhost:8081`.

#### Endpoints

See the endpoints at [api.yaml](./internal/api/api.go).

## Features

- Allow mount nzb files as the original file
- Allow streaming of video files
- Allow upload new files full obfuscated to prevent DMCA takedowns
- Filesystem upload new files to usenet automatically
- Api to manage the server

## Usage

### Download

Choose the release for your system at [releases](https://github.com/javi11/usenet-drive/releases).

### Run

Create the required folders:

```bash
mkdir -p ./config ./nzbs
```

Create the config file:

```bash
nano ./config/config.yaml
```

See an example at [config.yaml](config.sample.yaml).

Run the application:

```bash
./usenet-drive -c ./config/config.yaml
```

## Docker usage

### Run

Create the required folders:

```bash
mkdir -p ./config ./nzbs
```

Create a config file:

```bash
cp config.sample.yaml ./config/config.yaml
```

Edit the config file:

```bash
nano ./config/config.yaml
```

Run docker compose:

```bash
docker-compose up
```

Alternatively, you can run the docker image directly creating a docker compose:

```yaml
version: "3"
services:
  usenet-drive:
    image: laris11/usenet-drive:latest
    command: /usenet-drive -c /config/config.yaml
    ports:
      - "8080:8080"
    volumes:
      - ./config:/config
      - ./nzbs:/nzbs
    environment:
      - PUID=1000
      - PGID=1000
    restart: unless-stopped
```

## Config Struct

The `Config` struct defines the configuration for the Usenet Drive application. See an example at [config.yaml](config.example.toml).

### Fields

- `nzb_cache_size` (int): The number of NZBs to keep in memory. Default value is `100`. WARN remember that each NZB can be a big file increasing this will increase the memory usage.
- `root_path` (string!): The root path of your webdav virtual file system and where all nzb and not uploaded files will be saved. It is recommended to add a path to a fast disk for instance a SSD or NVME since this will improve a lot the playback of video files.
- `web_dav_port` (string): The port number for the server. Default value is `8080`.
- `api_port` (string): The port number for the server. Default value is `8080`.
- `usenet` (Usenet): The Usenet configuration.
- `db_path` (string): The path where the database will be saved. Default value is `/config/usenet-drive.db`.
- `rclone` (Rclone): The Rclone configuration.

## Rclone Struct

Since rclone webdav backend do not supports polling, if an vfs controller url is provided, the application will refresh the rclone cache when a new file is uploaded.

### Fields

- `vfs_url` (string): The url+port to the rclone vfs . Example `http://localhost:7579`.

## Usenet Struct

The `usenet` struct defines the Usenet configuration.

### Fields

- `download` (UsenetProvider): The Usenet provider for downloading.
- `upload` (Upload): The Usenet options for uploading.

## Upload Struct

The `Upload` struct defines the Usenet provider for uploading.

### Fields

- `provider` (UsenetProvider): Usenet provider to upload files
  Alternatively, you can use the same provider and split the available connections to allow more parallel uploads.
- `file_allow_list` ([]string): The list of allowed file extensions. For example, `[".mkv", ".mp4"]`, in this case only files with the extensions `.mkv` and `.mp4` will be uploaded to usenet. Take care not upload files that change frequently, like subtitules or text files, since they will be uploaded every time they change. In usenet you can not edit files. **_If using rclone crypt all file extensions will ends with .bin so in order to specify the real extension, you must add .bin at the end. Ex: .mkv.bin ._**

## UsenetProvider Struct

The `UsenetProvider` struct defines the Usenet provider configuration.

### Fields

- `host` (string): The hostname of the Usenet provider. For example, `news.usenetserver.com`.
- `port` (int): The port number of the Usenet provider. For example, `563`.
- `username` (string): The username for the Usenet provider. For example, `user`.
- `password` (string): The password for the Usenet provider. For example, `pass`.
- `groups` ([]string): The list of Usenet groups. For example, `["alt.binaries.teevee", "alt.binaries.movies"]`.
- `ssl` (bool): Whether to use SSL for the Usenet provider. Default value is `true`.
- `max_connections` (int): The maximum number of connections to the Usenet provider.

## Limitations

- Files uploaded to usenet can not be edited. If you need to edit a file, you need to upload a new file with the changes. This is more a limitation of usenet itself than the tool. (Future workaround can be done)
- The number of reads by file is limited by the number of connections to the usenet provider, normally not more than 3 connections are needed peer file read.

## Profiling

```bash
go tool pprof -http=:8082 http://localhost:8080/debug/pprof/profile
```
