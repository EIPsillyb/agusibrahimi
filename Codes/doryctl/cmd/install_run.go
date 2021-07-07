package cmd

import (
	"fmt"
	"github.com/dorystack/doryctl/pkg"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

type OptionsInstallRun struct {
	*OptionsCommon
	Mode     string
	FileName string
	Stdin    []byte
}

func NewOptionsInstallRun() *OptionsInstallRun {
	var o OptionsInstallRun
	o.OptionsCommon = OptCommon
	return &o
}

func NewCmdInstallRun() *cobra.Command {
	o := NewOptionsInstallRun()

	msgUse := fmt.Sprintf("run")
	msgShort := fmt.Sprintf("run install dory-core with docker or kubernetes")
	msgLong := fmt.Sprintf(`run install dory-core and relative components with docker-compose or kubernetes`)
	msgExample := fmt.Sprintf(`# run install dory-core and relative components with docker-compose, will create all config files and docker-compose.yaml file
%s install run --mode docker -f docker.yaml

#  run install dory-core and relative components with kubernetes, will create all config files and kubernetes deploy YAML files
%s install run --mode kubernetes -f kubernetes.yaml
`, pkg.BaseCmdName, pkg.BaseCmdName)

	cmd := &cobra.Command{
		Use:                   msgUse,
		DisableFlagsInUseLine: true,
		Short:                 msgShort,
		Long:                  msgLong,
		Example:               msgExample,
		Run: func(cmd *cobra.Command, args []string) {
			cobra.CheckErr(o.Complete(cmd))
			cobra.CheckErr(o.Validate(args))
			cobra.CheckErr(o.Run(args))
		},
	}
	cmd.Flags().StringVar(&o.Mode, "mode", "", "install mode, options: docker, kubernetes")
	cmd.Flags().StringVarP(&o.FileName, "file", "f", "", "install settings YAML file")
	return cmd
}

func (o *OptionsInstallRun) Complete(cmd *cobra.Command) error {
	var err error
	return err
}

func (o *OptionsInstallRun) Validate(args []string) error {
	var err error
	if o.Mode != "docker" && o.Mode != "kubernetes" {
		err = fmt.Errorf("[ERROR] --mode must be docker or kubernetes")
		return err
	}
	if o.FileName == "" {
		err = fmt.Errorf("[ERROR] -f required")
		return err
	}
	if o.FileName == "-" {
		bs, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		o.Stdin = bs
		if len(o.Stdin) == 0 {
			err = fmt.Errorf("[ERROR] -f - required os.stdin\n example: echo 'xxx' | %s install run --mode %s --f -", pkg.BaseCmdName, o.Mode)
			return err
		}
	}
	return err
}

// Run executes the appropriate steps to run a model's documentation
func (o *OptionsInstallRun) Run(args []string) error {
	var err error

	bs := []byte{}

	if o.FileName == "-" {
		bs = o.Stdin
	} else {
		bs, err = os.ReadFile(o.FileName)
		if err != nil {
			return err
		}
	}

	if o.Mode == "docker" {
		defer os.RemoveAll(pkg.DirInstallScripts)

		var idc pkg.InstallDockerConfig
		err = yaml.Unmarshal(bs, &idc)
		if err != nil {
			return err
		}
		validate := validator.New()
		err = validate.Struct(idc)
		if err != nil {
			return err
		}

		vals := map[string]interface{}{}
		err = yaml.Unmarshal(bs, &vals)
		if err != nil {
			return err
		}

		scriptName := fmt.Sprintf("%s/dory/docker/docker_certs.sh", pkg.DirInstallScripts)
		bs, err = pkg.FsInstallScripts.ReadFile(scriptName)
		if err != nil {
			return err
		}
		strScript, err := pkg.ParseTplFromVals(vals, string(bs))
		if err != nil {
			return err
		}

		_ = os.MkdirAll(fmt.Sprintf("%s/docker", pkg.DirInstallScripts), 0700)
		err = os.WriteFile(scriptName, []byte(strScript), 0600)
		if err != nil {
			return err
		}

		LogInfo("create docker certificates begin")
		_, _, err = pkg.CommandExec(fmt.Sprintf("sh %s/dory/docker/docker_certs.sh", pkg.DirInstallScripts), ".")
		if err != nil {
			err = fmt.Errorf("create docker certificates error: %s", err.Error())
			LogError(err.Error())
			return err
		}
		LogSuccess("create docker certificates success")
	} else if o.Mode == "kubernetes" {
		fmt.Println("args:", args)
		fmt.Println("install with kubernetes")
	}
	return err
}