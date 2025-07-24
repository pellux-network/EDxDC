package conf

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows/registry"
	"gopkg.in/yaml.v2"
)

// Conf is the app config
type Conf struct {
	JournalsFolder  string          `yaml:"journalsfolder"`
	Pages           map[string]bool `yaml:"pages"`
	CheckForUpdates bool            `yaml:"checkforupdates"`
	Loglevel        string          `json:"loglevel" yaml:"loglevel"`
}

// LoadOrCreateConf loads the config from the given path, or creates a default one if missing.
func LoadOrCreateConf(confPath string) Conf {
	log.Debug().Msg("Loading configuration...")
	defer log.Debug().Msg("Configuration loaded.")

	// If config does not exist, create it with defaults
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		log.Warn().Str("path", confPath).Msg("Config file not found, creating default config.")
		defaultYAML := `journalsfolder: "%USERPROFILE%\\Saved Games\\Frontier Developments\\Elite Dangerous"

pages:
  destination: true
  location: true
  cargo: true

checkforupdates: true
loglevel: info
`
		if err := os.MkdirAll(filepath.Dir(confPath), 0755); err != nil {
			log.Fatal().Err(err).Msg("Failed to create config directory")
			os.Exit(1)
		}
		if err := os.WriteFile(confPath, []byte(defaultYAML), 0644); err != nil {
			log.Fatal().Err(err).Msg("Failed to write default config")
			os.Exit(1)
		}
		var conf Conf
		if err := yaml.Unmarshal([]byte(defaultYAML), &conf); err != nil {
			log.Fatal().Err(err).Msg("Failed to unmarshal default config")
			os.Exit(1)
		}
		return conf
	}

	data, err := os.ReadFile(confPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read config file")
		os.Exit(1)
	}
	var conf Conf
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to unmarshal config file")
		os.Exit(1)
	}

	return conf
}

// ExpandJournalFolderPath expands any env variables in the journal folder path.
func (c Conf) ExpandJournalFolderPath() string {
	exp, _ := registry.ExpandString(c.JournalsFolder)
	return exp
}
