package config

import (
	"os"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v2"
)

type Config struct {
	LogPath    string `yaml:"log_path" default:"/config/activity.log"`
	RootPath   string `yaml:"root_path"`
	WebDavPort string `yaml:"web_dav_port" default:"8080"`
	ApiPort    string `yaml:"api_port" default:"8081"`
	Usenet     Usenet `yaml:"usenet"`
	DBPath     string `yaml:"db_path" default:"/config/usenet-drive.db"`
	Rclone     Rclone `yaml:"rclone"`
	Debug      bool   `yaml:"debug" default:"false"`
}

type Rclone struct {
	VFSUrl string `yaml:"vfs_url"`
}

type Usenet struct {
	Download           Download `yaml:"download"`
	Upload             Upload   `yaml:"upload"`
	FakeConnections    bool     `yaml:"fake_connections" default:"false"`
	ArticleSizeInBytes int64    `yaml:"article_size_in_bytes" default:"750000"`
}

type Download struct {
	MaxAheadDownloadSegments int              `yaml:"max_ahead_download_segments"`
	MaxRetries               int              `yaml:"max_retries" default:"8"`
	MaxCacheSizeInMB         int              `yaml:"max_cache_size_in_mb" default:"512"`
	Providers                []UsenetProvider `yaml:"providers"`
}

type Upload struct {
	DryRun        bool             `yaml:"dry_run" default:"false"`
	FileAllowlist []string         `yaml:"file_allow_list"`
	MaxRetries    int              `yaml:"max_retries" default:"8"`
	Groups        []string         `yaml:"groups"`
	Providers     []UsenetProvider `yaml:"providers"`
}

type UsenetProvider struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password" json:"-"`
	TLS            bool   `yaml:"tls"`
	MaxConnections int    `yaml:"max_connections"`
	InsecureSSL    bool   `yaml:"insecure_ssl" default:"false"`
	JoinGroup      bool   `yaml:"join_group" default:"false"`
}

func FromFile(path string) (*Config, error) {
	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse the config file
	var config Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		return nil, err
	}

	err = defaults.Set(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
