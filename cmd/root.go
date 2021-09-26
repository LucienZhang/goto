package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

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
					name := strings.Replace(strings.ToLower(conf.Commands[index].Name), " ", "", -1)
					input = strings.Replace(strings.ToLower(input), " ", "", -1)
					return strings.Contains(name, input)
				},
				StartInSearchMode: conf.StartInSearchMode,
			}
			idx, _, err := prompt.Run()
			cobra.CheckErr(err)
			shellCmd := exec.Command(conf.Shell, "-c", conf.Commands[idx].Cmd)
			shellCmd.Stdin = os.Stdin
			shellCmd.Stdout = os.Stdout
			shellCmd.Stderr = os.Stderr
			return shellCmd.Run()
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
