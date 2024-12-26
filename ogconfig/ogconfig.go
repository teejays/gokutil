package ogconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/teejays/gokutil/naam"
	"github.com/teejays/gokutil/panics"
	"gopkg.in/yaml.v3"
)

type Config struct {
	CLIConfig
	FileConfig
	CLIOrFileConfig
	InternalConfig
}

type CLIConfig struct {
	AppRootFromCurrDirPath string `arg:"-d,--app-dir" help:"The directory where the goku.yaml file is located" default:"."`
	CLIOrFileConfig
}

func (c CLIConfig) GetAppRootPathPath() string {
	return c.AppRootFromCurrDirPath
}

type HasAppRootPath interface {
	GetAppRootPathPath() string
}

type FileConfig struct {
	AppName                  naam.Name      `yaml:"app_name"`
	Description              string         `yaml:"description"`
	GoModuleName             string         `yaml:"go_module_name"`
	GokuVersionAtCreate      string         `yaml:"goku_version_at_create"`
	SchemaDirFromAppRootPath string         `yaml:"schema_dir"`
	Mods                     []ModReference `yaml:"mods"`
	CLIOrFileConfig          `yaml:",inline"`
}

type ModReference struct {
	Name    naam.Name  `yaml:"name"`
	Version ModVersion `yaml:"version"`
}

type ModVersion string

// CLIOrFileConfig are properties that can be set through the goku.yaml file but can be overwritten through the command line
type CLIOrFileConfig struct {
	PrintDebugFiles bool     `yaml:"print_debug_files" arg:"--print-debug-files" help:"print debug files"`
	Components      []string `yaml:"components" arg:"-c,--components" help:"comma separated list of components to generate. You can use the name of the component (snake-case) as defined in the component's code, or common names like 'all', 'backend', 'frontend', 'golang', 'graphql'"`
}

type InternalConfig struct {
	// Internal use only
	GokuVersionAtGenerate string          // The version of goku at the time of generating the code
	GokuUtilModuleName    string          // The name of the gokutil module
	ComponentsLookup      map[string]bool // This is a map of component name to bool. It is populated from EnabledComponents

	// Paths
	AppRootPath  Path
	SchemaPath   Path
	GoModulePath Path
}

type Path struct {
	FromApp     string // Relative to the app root dir
	FromCurrDir string // Relative to the current working directory
	Full        string // Full path
}

var _config Config
var configInitialized bool

func GetConfig() Config {
	panics.If(!configInitialized, "GetConfig() called before config is initialized")
	return _config
}

// SetConfig sets the config. This is useful for testing.
func SetConfig(cfg Config) {
	_config = cfg
	configInitialized = true
}

// UnsetConfig unsets the config. This is useful for testing. Always call this after calling SetConfig()
func UnsetConfig() {
	configInitialized = false
	_config = Config{}
}

func InitializeConfig(gokuVersion string, cli *CLIConfig) error {

	// Load File Config
	fileCfg, err := LoadAppGokuYaml(cli.AppRootFromCurrDirPath)
	if err != nil {
		return fmt.Errorf("loading goku.yaml file: %w", err)
	}

	return InitializeConfigCustomFileConfig(gokuVersion, cli, fileCfg)
}

func InitializeConfigCustomFileConfig(gokuVersion string, cli *CLIConfig, fileCfg FileConfig) error {

	if cli == nil {
		cli = &CLIConfig{}
	}

	// Command line only config
	if cli.AppRootFromCurrDirPath == "" {
		cli.AppRootFromCurrDirPath = "."
	}

	// Yaml and command line config: command line overwrites yaml
	components := cli.Components
	if len(components) < 1 {
		components = fileCfg.Components
	}
	printDebugFiles := cli.PrintDebugFiles
	if !printDebugFiles {
		printDebugFiles = fileCfg.PrintDebugFiles
	}

	// Internal config

	currDirPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current working directory: %w", err)
	}

	var internalConfig = InternalConfig{
		GokuVersionAtGenerate: gokuVersion,
		GokuUtilModuleName:    "github.com/teejays/gokutil",
		ComponentsLookup:      map[string]bool{}, // This is populated below
		AppRootPath: NewPath(NewPathReq{
			FromApp:         ".",
			AppRelToCurrDir: cli.AppRootFromCurrDirPath,
			CurrDir:         currDirPath,
		}),
		SchemaPath: NewPath(NewPathReq{
			FromApp:         fileCfg.SchemaDirFromAppRootPath,
			AppRelToCurrDir: cli.AppRootFromCurrDirPath,
			CurrDir:         currDirPath,
		}),
		GoModulePath: NewPath(NewPathReq{
			FromApp:         "backend",
			AppRelToCurrDir: cli.AppRootFromCurrDirPath,
			CurrDir:         currDirPath,
		}),
	}

	for _, v := range components {
		internalConfig.ComponentsLookup[v] = true
	}

	cfg := Config{
		CLIConfig:  *cli,
		FileConfig: fileCfg,
		CLIOrFileConfig: CLIOrFileConfig{
			PrintDebugFiles: printDebugFiles,
			Components:      components,
		},
		InternalConfig: internalConfig,
	}

	_config = cfg

	configInitialized = true
	return nil

}

func LoadAppGokuYaml(appRootDir string) (FileConfig, error) {

	f, err := os.Open(filepath.Join(appRootDir, "goku.yaml"))
	if err != nil {
		return FileConfig{}, fmt.Errorf("opening goku.yaml file: %w", err)
	}
	defer f.Close()

	var cgf FileConfig
	// Decode the YAML
	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)
	err = decoder.Decode(&cgf)
	if err != nil {
		return FileConfig{}, fmt.Errorf("Decoding YAML: %w", err)
	}

	return cgf, nil
}

type NewPathReq struct {
	FromApp         string
	AppRelToCurrDir string
	CurrDir         string
}

func NewPath(req NewPathReq) Path {
	appFullPath := filepath.Join(req.CurrDir, req.AppRelToCurrDir)
	fullPath := filepath.Join(appFullPath, req.FromApp)
	relPath, err := filepath.Rel(req.CurrDir, fullPath)
	panics.IfError(err, "getting relative path from base [%s] and target [%s]", req.CurrDir, fullPath)

	return Path{
		FromApp:     filepath.Clean(req.FromApp),
		FromCurrDir: filepath.Clean(relPath),
		Full:        filepath.Clean(fullPath),
	}
}
