package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"syscall"

	"github.com/LucienZhang/goto/configs"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
)

const (
	configFilePath = ".goto/.goto.yaml"
	esc            = "\033["
	foregroundCode = "38;2;"
)

type commandEntity struct {
	Name  string
	Desc  string
	Color string
	Cmd   string
}

func (c commandEntity) RGB(v interface{}) string {
	if c.Color == "" {
		return fmt.Sprintf("%v", v)
	}

	end := ""
	s, ok := v.(string)
	if !ok || !strings.HasSuffix(s, promptui.ResetCode) {
		end = promptui.ResetCode
	}
	return fmt.Sprintf("%s%s%sm%v%s", esc, foregroundCode, c.Color, v, end)
}

type config struct {
	Shell             string
	StartInSearchMode bool
	Commands          []commandEntity
}

var (
	conf      *config
	templates = promptui.SelectTemplates{
		Active:   fmt.Sprintf("%s {{ .Name | underline | .RGB }}", promptui.IconSelect),
		Inactive: "  {{ .Name | .RGB }}",
		Selected: fmt.Sprintf(`{{ "%s" | green }} Going to {{ .Name | .RGB }}`, promptui.IconGood),
		Details:  "{{ .Desc | faint }}",
	}
	rootCmd = &cobra.Command{
		Use:   "goto",
		Short: "Goto is an interactive command-line tool to manage your environments",
		Long: `An interactive command-line tool to manage your environments.
Complete documentation is available at https://github.com/LucienZhang/goto`,
		Version: configs.GetVersion(),
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := promptui.Select{
				Label:     "Select an environment to go",
				Items:     conf.Commands,
				Templates: &templates,
				Searcher: func(input string, index int) bool {
					// This algorithm is to check if input is a subsequence of command name, case insensitively.
					input = strings.ToLower(input)
					name := strings.ToLower(conf.Commands[index].Name)
					i, j := 0, 0
					for i < len(input) && j < len(name) {
						if input[i] == name[j] {
							i++
						}
						j++
					}
					return i == len(input)
				},
				StartInSearchMode: conf.StartInSearchMode,
			}
			idx, _, err := prompt.Run()
			cobra.CheckErr(err)
			shellCmd := exec.Command(conf.Shell, "-c", conf.Commands[idx].Cmd)
			return syscall.Exec(shellCmd.Path, argv(shellCmd), addCriticalEnv(dedupEnv(envv(shellCmd))))
		},
		DisableAutoGenTag: true,
	}
)

// Execute executes the root command.
func Execute() {
	err := rootCmd.Execute()
	cobra.CheckErr(err)
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	fullConfigFilePath := path.Join(home, configFilePath)
	viper.SetConfigFile(fullConfigFilePath)

	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(path.Dir(fullConfigFilePath), 0755)
			cobra.CheckErr(err)
			file, err := os.OpenFile(fullConfigFilePath, os.O_RDONLY|os.O_CREATE, 0644)
			cobra.CheckErr(err)
			file.Close()
		} else {
			cobra.CheckErr(err)
		}
	}

	conf = &config{"bash", false, []commandEntity{{"Help", "Show help information", "255;255;51", `echo 'Please config your commands in file ~/.goto/.goto.yaml.
Complete documentation is available at https://github.com/LucienZhang/goto'`}}}
	err = viper.Unmarshal(conf)
	cobra.CheckErr(err)
}

// GenManTree generates man page for the command.
func GenManTree(dir string) {
	header := &doc.GenManHeader{}
	err := doc.GenManTree(rootCmd, header, dir)
	if err != nil {
		log.Fatal(err)
	}
}

// GenMarkdownTree generates markdown doc for the command.
func GenMarkdownTree(dir string) {
	err := doc.GenMarkdownTree(rootCmd, dir)
	if err != nil {
		log.Fatal(err)
	}
}

/* Derived from exec.go */

func envv(c *exec.Cmd) []string {
	if c.Env != nil {
		return c.Env
	}
	return syscall.Environ()
}

func argv(c *exec.Cmd) []string {
	if len(c.Args) > 0 {
		return c.Args
	}
	return []string{c.Path}
}

// dedupEnv returns a copy of env with any duplicates removed, in favor of
// later values.
// Items not of the normal environment "key=value" form are preserved unchanged.
func dedupEnv(env []string) []string {
	return dedupEnvCase(runtime.GOOS == "windows", env)
}

// dedupEnvCase is dedupEnv with a case option for testing.
// If caseInsensitive is true, the case of keys is ignored.
func dedupEnvCase(caseInsensitive bool, env []string) []string {
	out := make([]string, 0, len(env))
	saw := make(map[string]int, len(env)) // key => index into out
	for _, kv := range env {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			out = append(out, kv)
			continue
		}
		k := kv[:eq]
		if caseInsensitive {
			k = strings.ToLower(k)
		}
		if dupIdx, isDup := saw[k]; isDup {
			out[dupIdx] = kv
			continue
		}
		saw[k] = len(out)
		out = append(out, kv)
	}
	return out
}

// addCriticalEnv adds any critical environment variables that are required
// (or at least almost always required) on the operating system.
// Currently this is only used for Windows.
func addCriticalEnv(env []string) []string {
	if runtime.GOOS != "windows" {
		return env
	}
	for _, kv := range env {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			continue
		}
		k := kv[:eq]
		if strings.EqualFold(k, "SYSTEMROOT") {
			// We already have it.
			return env
		}
	}
	return append(env, "SYSTEMROOT="+os.Getenv("SYSTEMROOT"))
}
