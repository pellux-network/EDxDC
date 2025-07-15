package conf

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/registry"
	"gopkg.in/yaml.v2"
)

// Conf is the app config
type Conf struct {
	JournalsFolder  string          `yaml:"journalsfolder"`
	Pages           map[string]bool `yaml:"pages"`
	CheckForUpdates bool            `yaml:"checkforupdates"`
}

// LoadOrCreateConf loads the config from the given path, or creates a default one if missing.
func LoadOrCreateConf(confPath string) Conf {
	log.Debugln("Loading configuration...")
	defer log.Debugln("Configuration loaded.")

	// If config does not exist, create it with defaults
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		log.Warnf("Config file not found at %s, creating default config.", confPath)
		defaultYAML := `journalsfolder: "%USERPROFILE%\\Saved Games\\Frontier Developments\\Elite Dangerous"

pages:
  destination: true
  location: true
  cargo: true

checkforupdates: true
`
		_ = os.MkdirAll(filepath.Dir(confPath), 0755)
		if err := os.WriteFile(confPath, []byte(defaultYAML), 0644); err != nil {
			log.Fatalln("Failed to write default config:", err)
		}
		var conf Conf
		if err := yaml.Unmarshal([]byte(defaultYAML), &conf); err != nil {
			log.Fatalln("Failed to unmarshal default config:", err)
		}
		return conf
	}

	data, err := os.ReadFile(confPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	var conf Conf
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatalln(err)
	}

	return conf
}

// ExpandJournalFolderPath expands any env variables in the journal folder path.
func (c Conf) ExpandJournalFolderPath() string {
	exp, _ := registry.ExpandString(c.JournalsFolder)
	return exp
}
