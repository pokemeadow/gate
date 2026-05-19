package pokeperms

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the main database, aliases, and localization settings
type Config struct {
	StorageType string   `yaml:"storage-type"` // "mysql" or "sqlite"
	Aliases     []string `yaml:"aliases"`      // Command aliases like ["pp", "perms"]
	Admins      []string `yaml:"admins"`       // Proxy-level bypass administrators (Usernames)
	Database    struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Name     string `yaml:"name"`
	} `yaml:"database"`
	Messages Messages `yaml:"messages"` // Full translation and localization node
}

// Messages handles all user-facing strings in the proxy layer
type Messages struct {
	Prefix             string `yaml:"prefix"`
	NoPermission       string `yaml:"no-permission"`
	InvalidDuration    string `yaml:"invalid-duration"`
	PlayerNotFound     string `yaml:"player-not-found"`
	UserCleared        string `yaml:"user-cleared"`
	UserParentSet      string `yaml:"user-parent-set"`
	UserParentAddTemp  string `yaml:"user-parent-addtemp"`
	UserPermSet        string `yaml:"user-perm-set"`
	UserPermUnset      string `yaml:"user-perm-unset"`
	GroupCreated       string `yaml:"group-created"`
	GroupDeleted       string `yaml:"group-deleted"`
	GroupPrefixSet     string `yaml:"group-prefix-set"`
	GroupWeightSet     string `yaml:"group-weight-set"`
	GroupParentAdded   string `yaml:"group-parent-added"`
	GroupParentRemoved string `yaml:"group-parent-removed"`
	GroupPermSet       string `yaml:"group-perm-set"`
	GroupPermUnset     string `yaml:"group-perm-unset"`
}

// DefaultConfig generates the standard customizable template for config.yml
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.StorageType = "sqlite"
	cfg.Aliases = []string{"pp", "pokepermissions"}

	// আপনার ইউজারনেম ডিফল্ট অ্যাডমিন হিসেবে সেট করা হলো
	cfg.Admins = []string{"ifte_"}

	cfg.Database.Host = "127.0.0.1"
	cfg.Database.Port = 3306
	cfg.Database.User = "root"
	cfg.Database.Password = "password"
	cfg.Database.Name = "pokeperms"

	// Default messages setup
	cfg.Messages.Prefix = "&8[&bPokePerms&8] "
	cfg.Messages.NoPermission = "&cYou do not have permission to execute this command!"
	cfg.Messages.InvalidDuration = "&cInvalid duration format! Use standard format like 30d, 12h, 45m."
	cfg.Messages.PlayerNotFound = "&cTarget player database reference error or offline."
	cfg.Messages.UserCleared = "&aSuccessfully cleared all permission data for &e%s&a."
	cfg.Messages.UserParentSet = "&aSet &e%s's &aprimary group to &b%s&a."
	cfg.Messages.UserParentAddTemp = "&aAdded temporary group &b%s &ato &e%s &afor &d%s&a."
	cfg.Messages.UserPermSet = "&aSet permission &b%s &ato &d%t &afor &e%s &8(%s)"
	cfg.Messages.UserPermUnset = "&aUnset permission &b%s &afor &e%s &8(%s)"
	cfg.Messages.GroupCreated = "&aGroup &b%s &acreated successfully."
	cfg.Messages.GroupDeleted = "&aGroup &b%s &adeleted successfully."
	cfg.Messages.GroupPrefixSet = "&aSet prefix of group &b%s &ato: %s"
	cfg.Messages.GroupWeightSet = "&aSet weight of group &b%s &ato &d%d&a."
	cfg.Messages.GroupParentAdded = "&aGroup &b%s &anow inherits permissions from parent &d%s&a."
	cfg.Messages.GroupParentRemoved = "&aGroup &b%s &ano longer inherits from parent &d%s&a."
	cfg.Messages.GroupPermSet = "&aSet group permission &b%s &ato &d%t &afor group &e%s &8(%s)"
	cfg.Messages.GroupPermUnset = "&aUnset group permission &b%s &afor group &e%s &8(%s)"

	return cfg
}

// LoadConfig reads the config file or creates it with defaults
func LoadConfig(dir string) (*Config, error) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "config.yml")

	// ফাইলের বর্তমান অবস্থা চেক করা হচ্ছে
	info, err := os.Stat(path)

	// ফাইল যদি না থাকে, অথবা ফাইলের সাইজ যদি ১৫ বাইটের কম হয় (যেমন খালি বা '404: Not Found')
	if os.IsNotExist(err) || (err == nil && info.Size() <= 15) {
		cfg := DefaultConfig()
		d, _ := yaml.Marshal(cfg)
		if err := os.WriteFile(path, d, 0666); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	// ফাইল রিড করার সময় যদি YAML ফরম্যাট ভুল পায়, তবে ক্র্যাশ না করে নতুন করে ডিফল্ট কনফিগ লিখে দেবে
	if err := yaml.Unmarshal(data, cfg); err != nil {
		d, _ := yaml.Marshal(cfg)
		_ = os.WriteFile(path, d, 0666)
		return cfg, nil
	}

	// সেফটি চেক: যদি ফাইল আনমার্শাল হলেও মেইন ডেটা খালি থাকে
	if cfg.StorageType == "" {
		cfg = DefaultConfig()
		d, _ := yaml.Marshal(cfg)
		_ = os.WriteFile(path, d, 0666)
	}

	return cfg, nil
}