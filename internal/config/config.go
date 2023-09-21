package config

import (
	"os"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v2"
)

type Config struct {
	LogPath      string `yaml:"log_path" default:"/config/activity.log"`
	TmpPath      string `yaml:"tmp_path"`
	RootPath     string `yaml:"root_path"`
	WebDavPort   string `yaml:"web_dav_port" default:"8080"`
	ApiPort      string `yaml:"api_port" default:"8081"`
	Usenet       Usenet `yaml:"usenet"`
	DBPath       string `yaml:"db_path" default:"/config/usenet-drive.db"`
	NzbCacheSize int    `yaml:"nzb_cache_size" default:"100"`
	Rclone       Rclone `yaml:"rclone"`
}

type Rclone struct {
	VFSUrl string `yaml:"vfs_url"`
}

type Usenet struct {
	Download UsenetProvider `yaml:"download"`
	Upload   Upload         `yaml:"upload"`
}

type Upload struct {
	DryRun                  bool             `yaml:"dry_run" default:"false"`
	Providers               []UsenetProvider `yaml:"providers"`
	FileAllowlist           []string         `yaml:"file_allow_list"`
	NyuuVersion             string           `yaml:"nyuu_version" default:"0.4.1"`
	NyuuPath                string           `yaml:"nyuu_path" default:"/config/nyuu"`
	UploadIntervalInSeconds float64          `yaml:"upload_interval_in_seconds" default:"60"`
}

type UsenetProvider struct {
	Host           string   `yaml:"host"`
	Port           int      `yaml:"port"`
	Username       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	Groups         []string `yaml:"groups"`
	SSL            bool     `yaml:"ssl"`
	MaxConnections int      `yaml:"max_connections"`
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

	defaults.Set(&config)

	return &config, nil
}
