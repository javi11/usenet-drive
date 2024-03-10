# usenet-drive (WIP)

[![Tests](https://github.com/javi11/usenet-drive/actions/workflows/test.yaml/badge.svg)](https://github.com/javi11/usenet-drive/actions/workflows/test.yaml)

<a href="https://www.buymeacoffee.com/qbt52hh7sjd"><img src="https://img.buymeacoffee.com/button-api/?text=Buy me a coffee&emoji=☕&slug=qbt52hh7sjd&button_colour=FFDD00&font_colour=000000&font_family=Comic&outline_colour=000000&coffee_colour=ffffff" /></a>

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
    command: /usenetdrive -c /config/config.yaml
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

- `download` (Download): Downloading configuration.
- `upload` (Upload): Uploading configuration.
- `providers` (UsenetProvider): Usenet providers to download/upload files
- `fake_connections` (bool): Whether to use fake connections. Default value is `false`. This is useful for testing purposes.
- `max_connection_ttl_in_minutes` (int): The maximum time a connection will be kept alive in minutes. Default value is `60`.
- `max_connection_idle_time_in_minutes` (int): Idle connections will be closed after this max time `10`.

## Download Struct

The `Download` struct defines the Usenet provider for downloading.

### Fields

- `max_download_workers` (int): The maximum number of download workers. Default value is `15`. WARN the tool will use 1 connections per worker. Min value is 1. The number observed optimal for good speed is 15 depending on the number of connections, if you add more download workers than connections won't improve the speed. You must have a balance between the max_download_workers, the max_connections and the number of parallel opened files. For instance, you can have 30 connections and 30 download workers peer file, in this case one file opened for download will use the full power of the download connections achieving good speeds, but you will probably have problems if you try to open more than 1 file since connections will be split between the opened files.
- `max_retries` (int): The maximum number of retries to download a segment. Default value is `8`.
- `providers` (UsenetProvider): Usenet providers to download files. (It is recommended an unlimited provider for this)
- `max_buffer_size_in_mb` (int): The maximum memory buffer size in MB. Default value is `512`. This is the maximum size of the chunks that will be stored in memory on downloads. This value will be multiplied by 2 to fulfil the number of shards. Also, this value can not be lower than the `article_size_in_bytes * 2` or the nzb with the bigger article size multiplied by 2. If this value is to low compared off the `max_download_workers` you won't be able to achieve good speeds since there will not be space to save chunks at the cache. For instance, if we use the default vale `512` the max consumption on memory will be ~1024MB.

## Upload Struct

The `Upload` struct defines the Usenet provider for uploading.

### Fields

- `file_allow_list` ([]string): The list of allowed file extensions. For example, `[".mkv", ".mp4"]`, in this case only files with the extensions `.mkv` and `.mp4` will be uploaded to usenet. Take care not upload files that change frequently, like subtitules or text files, since they will be uploaded every time they change. In usenet you can not edit files. **_If using rclone crypt all file extensions will ends with .bin so in order to specify the real extension, you must add .bin at the end. Ex: .mkv.bin ._**
- `max_retries` (int): The maximum number of retries to upload a segment. Default value is `8`.
- `providers` (UsenetProvider): Usenet providers to upload files. (It is recommended a block account for this)

## UsenetProvider Struct

The `UsenetProvider` struct defines the Usenet provider configuration.

### Fields

- `host` (string): The hostname of the Usenet provider. For example, `news.usenetserver.com`.
- `port` (int): The port number of the Usenet provider. For example, `563`.
- `username` (string): The username for the Usenet provider. For example, `user`.
- `password` (string): The password for the Usenet provider. For example, `pass`.
- `groups` ([]string): The list of Usenet groups. For example, `["alt.binaries.teevee", "alt.binaries.movies"]`.
- `tls` (bool): Whether to use SSL for the Usenet provider. Default value is `true`.
- `max_connections` (int): The maximum number of connections to the Usenet provider.
- `download_only` (bool): Whether this provider only allows downloading. Default value is `false`.

## Limitations

- Files uploaded to usenet can not be edited. If you need to edit a file, you need to upload a new file with the changes. This is more a limitation of usenet itself than the tool. (Future workaround can be done)
- The number of reads by file is limited by the number of connections to the usenet provider, normally not more than 3 connections are needed peer file read.

## Profiling

```bash
go tool pprof -http=:8082 http://localhost:8080/debug/pprof/profile
```

## Highly inspired by

- Upload feature: https://github.com/F4n4t/GoPostStuff
- Yenc encode: https://github.com/F4n4t/GoPostStuff
- Nzb feature: https://github.com/chrisfarms/nzb
