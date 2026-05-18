package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// DefaultMask is used when config does not specify a mask.
const DefaultMask = "***"

// Config is the YAML-backed user configuration.
type Config struct {
	UseDefaults *bool    `yaml:"use_defaults"`
	Mask        *string  `yaml:"mask"`
	Names       []string `yaml:"names"`
	Patterns    []string `yaml:"patterns"`
}

// Effective is the fully resolved rule set after applying defaults.
type Effective struct {
	UseDefaults      bool
	Mask             string
	Fields           []Rule
	FieldPatterns    []Rule
	URLParams        []Rule
	URLParamPatterns []Rule
}

// Rule is a redaction rule value and its origin, such as "default" or "user".
type Rule struct{ Value, Source string }

var defaultFields = []string{"authorization", "proxy-authorization", "x-api-key", "api-key", "api_key", "client_secret", "client-secret", "access_token", "refresh_token", "id_token", "password", "passwd", "private_key", "private-key", "secret_key", "secret-key", "aws_secret_access_key"}
var defaultURLParams = []string{"access_token", "refresh_token", "id_token", "api_key", "apikey", "client_secret", "token"}

// DefaultPath returns the platform-specific config file path.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "redact", "config.yaml"), nil
}

// Load reads and validates config from path.
func Load(path string, explicit bool) (Config, bool, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if explicit {
			return Config{}, false, fmt.Errorf("config file not found: %s", path)
		}
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, true, err
	}
	return cfg, true, Validate(cfg)
}

// Validate checks config values before they are used by the redactor.
func Validate(cfg Config) error {
	if cfg.Mask != nil {
		if *cfg.Mask == "" {
			return fmt.Errorf("mask cannot be empty")
		}
		for _, r := range *cfg.Mask {
			if r == '\n' || r == '\r' || unicode.IsControl(r) {
				return fmt.Errorf("mask contains control character")
			}
		}
	}
	for _, name := range cfg.Names {
		if _, err := NormalizeName(name); err != nil {
			return err
		}
	}
	for _, p := range cfg.Patterns {
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("invalid regex %q: %w", p, err)
		}
	}
	return nil
}

// EffectiveRules combines default rules and user config into a redaction rule set.
func EffectiveRules(cfg Config) Effective {
	use := true
	if cfg.UseDefaults != nil {
		use = *cfg.UseDefaults
	}
	mask := DefaultMask
	if cfg.Mask != nil {
		mask = *cfg.Mask
	}
	e := Effective{UseDefaults: use, Mask: mask}
	if use {
		for _, v := range defaultFields {
			e.Fields = append(e.Fields, Rule{Value: v, Source: "default"})
		}
		for _, v := range defaultURLParams {
			e.URLParams = append(e.URLParams, Rule{Value: v, Source: "default"})
		}
	}
	for _, v := range cfg.Names {
		rule := Rule{Value: strings.ToLower(v), Source: "user"}
		e.Fields = append(e.Fields, rule)
		e.URLParams = append(e.URLParams, rule)
	}
	for _, v := range cfg.Patterns {
		rule := Rule{Value: v, Source: "user"}
		e.FieldPatterns = append(e.FieldPatterns, rule)
		e.URLParamPatterns = append(e.URLParamPatterns, rule)
	}
	return e
}

// NormalizeName canonicalizes an exact field or URL parameter name.
func NormalizeName(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", errors.New("name cannot be empty")
	}
	if strings.ContainsAny(s, ":=\"'?& 	\n\r") {
		return "", fmt.Errorf("invalid name: %q", s)
	}
	return s, nil
}

// ContainsRule reports whether rules contains v.
func ContainsRule(rules []Rule, v string) bool {
	return slices.ContainsFunc(rules, func(r Rule) bool { return r.Value == v })
}
