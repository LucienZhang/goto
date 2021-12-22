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
	"github.com/google/shlex"
	"github.com/manifoldco/promptui"
	"github.com/riywo/loginshell"
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
	Name     string
	Desc     string
	Color    string
	Cmd      string
	Shell    string
	ExecMode bool
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
	StartInSearchMode bool
	Commands          []commandEntity
}

var (
	conf      = &config{}
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
			currCmd := &conf.Commands[idx]
			if strings.Replace(currCmd.Cmd, " ", "", -1) == "" {
				return fmt.Errorf("command %s is empty", currCmd.Name)
			}
			var cmdToExec *exec.Cmd
			if currCmd.ExecMode {
				currCmdArgs, err := shlex.Split(currCmd.Cmd)
				cobra.CheckErr(err)
				cmdToExec = exec.Command(currCmdArgs[0], currCmdArgs[1:]...)
			} else {
				wrapperShell := currCmd.Shell
				if wrapperShell == "" {
					wrapperShell, err = loginshell.Shell()
					cobra.CheckErr(err)
				}
				cmdToExec = exec.Command(wrapperShell, "-c", currCmd.Cmd)
			}
			return syscall.Exec(cmdToExec.Path, argv(cmdToExec), addCriticalEnv(dedupEnv(envv(cmdToExec))))
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
	viper.SetDefault("StartInSearchMode", false)
	viper.SetDefault("Commands", []commandEntity{{"Help", "Show help information", "255;255;51", `echo 'Please config your commands in file ~/.goto/.goto.yaml.
	Complete documentation is available at https://github.com/LucienZhang/goto'`, "", false}})

	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(path.Dir(fullConfigFilePath), 0755)
			cobra.CheckErr(err)
			// file, err := os.OpenFile(fullConfigFilePath, os.O_RDONLY|os.O_CREATE, 0644)
			// file.Close()
			err = viper.SafeWriteConfigAs(fullConfigFilePath)
			cobra.CheckErr(err)
		} else {
			cobra.CheckErr(err)
		}
	}

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
