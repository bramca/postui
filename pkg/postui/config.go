package postui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Keybindings struct {
	Help                 string
	NextView             string
	PrevView             string
	NextTab              string
	PrevTab              string
	Up                   string
	Down                 string
	Left                 string
	Right                string
	ScrollUpMulti        string
	ScrollDownMulti      string
	GoDown               string
	GoUp                 string
	GoForward            string
	GoBack               string
	Top                  string
	Bottom               string
	Paste                string
	Copy                 string
	CopyCurl             string
	Save                 string
	Run                  string
	AddCollection        string
	FocusCollection      string
	ToggleCollectionEdit string
	ReloadConfig         string
	Quit                 string
}

type Color struct {
	Light string
	Dark  string
}

type Colors struct {
	Highlight    Color `mapstructure:"highlight-colors"`
	Nonhighlight Color `mapstructure:"non-highlight-colors"`
	ResponseTime Color `mapstructure:"response-time-colors"`
	ResponseSize Color `mapstructure:"response-size-colors"`
}

type Config struct {
	DefaultCollectionDir string      `mapstructure:"default-collection-dir"`
	DefaultKeybindings   Keybindings `mapstructure:"default-keybindings"`
	DefaultColors        Colors      `mapstructure:"default-colors"`
}

func NewConfig() (Config, error) {
	conf := Config{}
	viper.SetConfigName("config")
	viper.AddConfigPath("$HOME/.config/postui/")

	_ = viper.ReadInConfig()

	// Default collection directory
	viper.SetDefault("default-collection-dir", "~/.config/postui/collections")

	// Default keybindings
	viper.SetDefault("default-keybindings.help", "ctrl+o")
	viper.SetDefault("default-keybindings.nextView", "tab")
	viper.SetDefault("default-keybindings.prevView", "shift+tab")
	viper.SetDefault("default-keybindings.nextTab", "alt+]")
	viper.SetDefault("default-keybindings.prevTab", "alt+[")
	viper.SetDefault("default-keybindings.up", "ctrl+k")
	viper.SetDefault("default-keybindings.down", "ctrl+j")
	viper.SetDefault("default-keybindings.left", "ctrl+h")
	viper.SetDefault("default-keybindings.right", "ctrl+l")
	viper.SetDefault("default-keybindings.scrollUpMulti", "ctrl+b")
	viper.SetDefault("default-keybindings.scrollDownMulti", "ctrl+f")
	viper.SetDefault("default-keybindings.goDown", "j")
	viper.SetDefault("default-keybindings.goUp", "k")
	viper.SetDefault("default-keybindings.goForward", "l")
	viper.SetDefault("default-keybindings.goBack", "h")
	viper.SetDefault("default-keybindings.top", "g")
	viper.SetDefault("default-keybindings.bottom", "G")
	viper.SetDefault("default-keybindings.paste", "ctrl+v")
	viper.SetDefault("default-keybindings.copy", "alt+c")
	viper.SetDefault("default-keybindings.copyCurl", "alt+x")
	viper.SetDefault("default-keybindings.save", "ctrl+s")
	viper.SetDefault("default-keybindings.run", "ctrl+r")
	viper.SetDefault("default-keybindings.addCollection", "alt+a")
	viper.SetDefault("default-keybindings.focusCollection", "alt+l")
	viper.SetDefault("default-keybindings.toggleCollectionEdit", "ctrl+t")
	viper.SetDefault("default-keybindings.reloadConfig", "alt+r")
	viper.SetDefault("default-keybindings.quit", "ctrl+c")

	// Default colors
	viper.SetDefault("default-colors.highlight-colors.Light", "#82aaff")
	viper.SetDefault("default-colors.highlight-colors.Dark", "#B191FF")
	viper.SetDefault("default-colors.non-highlight-colors.Light", "#B5B5B5")
	viper.SetDefault("default-colors.non-highlight-colors.Dark", "#535353")
	viper.SetDefault("default-colors.response-time-colors.Light", "#72acff")
	viper.SetDefault("default-colors.response-time-colors.Dark", "#c792ea")
	viper.SetDefault("default-colors.response-size-colors.Light", "#72acff")
	viper.SetDefault("default-colors.response-size-colors.Dark", "#c792ea")

	err := viper.Unmarshal(&conf)

	return conf, err
}

// ExpandPath resolves ~ to home dir and converts to absolute path
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Abs(path)
}

// AtomicWrite safely writes data to a file, creating parent dirs if needed
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	// Write to temp file in same directory (ensures same filesystem for rename)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		if err := os.Remove(tmpPath); err != nil {
			return fmt.Errorf("failed to remove temp file: %w", err)
		}
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		if err := os.Remove(tmpPath); err != nil {
			return fmt.Errorf("failed to remove temp file: %w", err)
		}
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	return nil
}
