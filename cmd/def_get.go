package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/dory-engine/dory-ctl/pkg"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"strings"
)

type OptionsDefGet struct {
	*OptionsCommon `yaml:"optionsCommon" json:"optionsCommon" bson:"optionsCommon" validate:""`
	ModuleNames    []string `yaml:"moduleNames" json:"moduleNames" bson:"moduleNames" validate:""`
	EnvNames       []string `yaml:"envNames" json:"envNames" bson:"envNames" validate:""`
	BranchNames    []string `yaml:"branchNames" json:"branchNames" bson:"branchNames" validate:""`
	StepNames      []string `yaml:"stepNames" json:"stepNames" bson:"stepNames" validate:""`
	Full           bool     `yaml:"full" json:"full" bson:"full" validate:""`
	Output         string   `yaml:"output" json:"output" bson:"output" validate:""`
	Param          struct {
		Kinds       []string `yaml:"kinds" json:"kinds" bson:"kinds" validate:""`
		ProjectName string   `yaml:"projectName" json:"projectName" bson:"projectName" validate:""`
		IsAllKind   bool     `yaml:"isAllKind" json:"isAllKind" bson:"isAllKind" validate:""`
	}
}

func NewOptionsDefGet() *OptionsDefGet {
	var o OptionsDefGet
	o.OptionsCommon = OptCommon
	return &o
}

func NewCmdDefGet() *cobra.Command {
	o := NewOptionsDefGet()

	defCmdKinds := []string{}
	for k, _ := range pkg.DefCmdKinds {
		defCmdKinds = append(defCmdKinds, k)
	}

	msgUse := fmt.Sprintf(`get [projectName] [kind],[kind]... [--output=json|yaml] [--module=moduleName1,moduleName2] [--env=envName1,envName2] [--branch=branchName1,branchName2] [--step=stepName1,stepName2]
  # kind options: %s`, strings.Join(defCmdKinds, " / "))
	msgShort := fmt.Sprintf("get project definitions")
	msgLong := fmt.Sprintf(`get project definitions in dory-core server`)
	msgExample := fmt.Sprintf(`  # get project definitions summary
  doryctl def get test-project1

  # get project all definitions
  doryctl def get test-project1 all --output=yaml

  # get project all definitions, and show in full version
  doryctl def get test-project1 all --output=yaml --full

  # get project build and package modules definitions
  doryctl def get test-project1 build,package

  # get project deploy modules definitions, and filter by moduleNames and envNames
  doryctl def get test-project1 deploy --module=tp1-go-demo,tp1-gin-demo --env=test

  # get project pipeline definitions, and filter by branchNames
  doryctl def get test-project1 pipeline --branch=develop,release

  # get project custom step modules definitions, and filter by envNames and stepNames
  doryctl def get test-project1 step --env=test --step=testApi`)

	cmd := &cobra.Command{
		Use:                   msgUse,
		DisableFlagsInUseLine: true,
		Short:                 msgShort,
		Long:                  msgLong,
		Example:               msgExample,
		Run: func(cmd *cobra.Command, args []string) {
			CheckError(o.Complete(cmd))
			CheckError(o.Validate(args))
			CheckError(o.Run(args))
		},
	}
	cmd.Flags().StringSliceVar(&o.ModuleNames, "module", []string{}, "filter project definitions items by moduleNames")
	cmd.Flags().StringSliceVar(&o.EnvNames, "env", []string{}, "filter project definitions by envNames")
	cmd.Flags().StringSliceVar(&o.BranchNames, "branch", []string{}, "filter project pipeline definitions by branchNames")
	cmd.Flags().StringSliceVar(&o.StepNames, "step", []string{}, "filter project definitions by stepNames")
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "output format (options: yaml / json)")
	cmd.Flags().BoolVar(&o.Full, "full", false, "output project definitions in full version, use with --output option")
	return cmd
}

func (o *OptionsDefGet) Complete(cmd *cobra.Command) error {
	var err error
	err = o.GetOptionsCommon()
	return err
}

func (o *OptionsDefGet) Validate(args []string) error {
	var err error
	if len(args) == 0 {
		err = fmt.Errorf("projectName required")
		return err
	}
	var projectName string
	var kinds, kindParams []string
	projectName = args[0]
	if len(args) > 1 {
		kindsStr := args[1]
		arr := strings.Split(kindsStr, ",")
		for _, s := range arr {
			a := strings.Trim(s, " ")
			if a != "" {
				kinds = append(kinds, a)
			}
		}
		var foundAll bool
		for _, kind := range kinds {
			var found bool
			for cmdKind, _ := range pkg.DefCmdKinds {
				if kind == cmdKind {
					found = true
					break
				}
			}
			if !found {
				defCmdKinds := []string{}
				for k, _ := range pkg.DefCmdKinds {
					defCmdKinds = append(defCmdKinds, k)
				}
				err = fmt.Errorf("kind %s format error: not correct, options: %s", kind, strings.Join(defCmdKinds, " / "))
				return err
			}
			if kind == "all" {
				foundAll = true
			}
			kindParams = append(kindParams, pkg.DefCmdKinds[kind])
		}
		if foundAll == true {
			o.Param.IsAllKind = true
		}
		o.Param.Kinds = kindParams
	}

	err = pkg.ValidateMinusNameID(projectName)
	if err != nil {
		err = fmt.Errorf("projectName %s format error: %s", projectName, err.Error())
		return err
	}

	for _, moduleName := range o.ModuleNames {
		err = pkg.ValidateMinusNameID(moduleName)
		if err != nil {
			err = fmt.Errorf("moduleName %s format error: %s", moduleName, err.Error())
			return err
		}
	}
	o.Param.ProjectName = projectName

	if o.Output != "" {
		if o.Output != "yaml" && o.Output != "json" {
			err = fmt.Errorf("--output must be yaml or json")
			return err
		}
	}
	return err
}

func (o *OptionsDefGet) Run(args []string) error {
	var err error

	bs, _ := pkg.YamlIndent(o)
	log.Debug(fmt.Sprintf("command options:\n%s", string(bs)))

	param := map[string]interface{}{}
	result, _, err := o.QueryAPI(fmt.Sprintf("api/cicd/projectDef/%s", o.Param.ProjectName), http.MethodGet, "", param, false)
	if err != nil {
		return err
	}
	project := pkg.ProjectOutput{}
	err = json.Unmarshal([]byte(result.Get("data.project").Raw), &project)
	if err != nil {
		return err
	}

	defKinds := []pkg.DefKind{}
	if len(o.Param.Kinds) == 0 {
		defKind := pkg.DefKind{
			Kind: "projectSummary",
			Metadata: pkg.Metadata{
				ProjectName: project.ProjectInfo.ProjectName,
				Labels:      map[string]string{},
			},
			Items: []interface{}{},
		}
		var branchNames []string
		for _, pipeline := range project.ProjectPipelines {
			branchNames = append(branchNames, pipeline.BranchName)
		}
		var envNames []string
		for _, pae := range project.ProjectAvailableEnvs {
			envNames = append(envNames, pae.EnvName)
		}
		def := pkg.ProjectSummary{
			BuildEnvs:       project.BuildEnvs,
			BuildNames:      project.BuildNames,
			CustomStepConfs: project.CustomStepConfs,
			PackageNames:    project.PackageNames,
			BranchNames:     branchNames,
			EnvNames:        envNames,
			NodePorts:       project.NodePorts,
		}
		defKind.Items = append(defKind.Items, def)
		defKinds = append(defKinds, defKind)
	} else {
		defKindProject := pkg.DefKind{
			Kind: "",
			Metadata: pkg.Metadata{
				ProjectName: project.ProjectInfo.ProjectName,
				Labels:      map[string]string{},
			},
			Items: []interface{}{},
		}
		if len(project.ProjectDef.BuildDefs) > 0 {
			defKind := defKindProject
			defKind.Kind = "buildDefs"
			for _, def := range project.ProjectDef.BuildDefs {
				var isShow bool
				if len(o.ModuleNames) == 0 {
					isShow = true
				} else {
					for _, moduleName := range o.ModuleNames {
						if moduleName == def.BuildName {
							isShow = true
							break
						}
					}
				}
				if isShow {
					defKind.Items = append(defKind.Items, def)
				}
			}
			defKinds = append(defKinds, defKind)
		}

		if len(project.ProjectDef.PackageDefs) > 0 {
			defKind := defKindProject
			defKind.Kind = "packageDefs"
			defKind.Status.ErrMsg = project.ProjectDef.ErrMsgPackageDefs
			for _, def := range project.ProjectDef.PackageDefs {
				var isShow bool
				if len(o.ModuleNames) == 0 {
					isShow = true
				} else {
					for _, moduleName := range o.ModuleNames {
						if moduleName == def.PackageName {
							isShow = true
							break
						}
					}
				}
				if isShow {
					defKind.Items = append(defKind.Items, def)
				}
			}
			defKinds = append(defKinds, defKind)
		}

		if len(project.ProjectAvailableEnvs) > 0 {
			paes := []pkg.ProjectAvailableEnv{}
			for _, pae := range project.ProjectAvailableEnvs {
				if len(o.EnvNames) == 0 {
					paes = append(paes, pae)
				} else {
					for _, envName := range o.EnvNames {
						if envName == pae.EnvName {
							paes = append(paes, pae)
							break
						}
					}
				}
			}
			for _, pae := range paes {
				if len(pae.DeployContainerDefs) > 0 {
					defKind := defKindProject
					defKind.Kind = "deployContainerDefs"
					defKind.Status.ErrMsg = pae.ErrMsgDeployContainerDefs
					defKind.Metadata.Labels = map[string]string{
						"envName": pae.EnvName,
					}
					for _, def := range pae.DeployContainerDefs {
						var isShow bool
						if len(o.ModuleNames) == 0 {
							isShow = true
						} else {
							for _, moduleName := range o.ModuleNames {
								if moduleName == def.DeployName {
									isShow = true
									break
								}
							}
						}
						if isShow {
							defKind.Items = append(defKind.Items, def)
						}
					}
					defKinds = append(defKinds, defKind)
				}
			}

			for _, pae := range paes {
				if len(pae.CustomStepDefs) > 0 {
					csds := pkg.CustomStepDefs{}
					for stepName, csd := range pae.CustomStepDefs {
						if len(o.StepNames) == 0 {
							csds[stepName] = csd
						} else {
							for _, name := range o.StepNames {
								if name == stepName {
									csds[stepName] = csd
									break
								}
							}
						}
					}
					for stepName, csd := range csds {
						defKind := defKindProject
						defKind.Kind = "customStepDef"
						var errMsg string
						for name, msg := range pae.ErrMsgCustomStepDefs {
							if name == stepName {
								errMsg = msg
							}
						}
						defKind.Status.ErrMsg = errMsg
						defKind.Metadata.Labels = map[string]string{
							"envName":    pae.EnvName,
							"stepName":   stepName,
							"enableMode": csd.EnableMode,
						}
						for _, csmd := range csd.CustomStepModuleDefs {
							var isShow bool
							if len(o.ModuleNames) == 0 {
								isShow = true
							} else {
								for _, moduleName := range o.ModuleNames {
									if moduleName == csmd.ModuleName {
										isShow = true
										break
									}
								}
							}
							if isShow {
								defKind.Items = append(defKind.Items, csmd)
							}
						}
						defKinds = append(defKinds, defKind)
					}
				}
			}
		}

		if len(project.ProjectDef.CustomStepDefs) > 0 {
			csds := pkg.CustomStepDefs{}
			for stepName, csd := range project.ProjectDef.CustomStepDefs {
				if len(o.StepNames) == 0 {
					csds[stepName] = csd
				} else {
					for _, name := range o.StepNames {
						if name == stepName {
							csds[stepName] = csd
							break
						}
					}
				}
			}
			for stepName, csd := range csds {
				defKind := defKindProject
				defKind.Kind = "customStepDef"
				var errMsg string
				for name, msg := range project.ProjectDef.ErrMsgCustomStepDefs {
					if name == stepName {
						errMsg = msg
					}
				}
				defKind.Status.ErrMsg = errMsg
				defKind.Metadata.Labels = map[string]string{
					"stepName":   stepName,
					"enableMode": csd.EnableMode,
				}
				for _, csmd := range csd.CustomStepModuleDefs {
					var isShow bool
					if len(o.ModuleNames) == 0 {
						isShow = true
					} else {
						for _, moduleName := range o.ModuleNames {
							if moduleName == csmd.ModuleName {
								isShow = true
								break
							}
						}
					}
					if isShow {
						defKind.Items = append(defKind.Items, csmd)
					}
				}
				defKinds = append(defKinds, defKind)
			}
		}

		if len(project.ProjectPipelines) > 0 {
			pps := []pkg.ProjectPipeline{}
			for _, pp := range project.ProjectPipelines {
				if len(o.BranchNames) == 0 {
					pps = append(pps, pp)
				} else {
					for _, branchName := range o.BranchNames {
						if branchName == pp.BranchName {
							pps = append(pps, pp)
							break
						}
					}
				}
			}
			for _, pp := range pps {
				defKind := defKindProject
				defKind.Kind = "pipelineDef"
				defKind.Status.ErrMsg = pp.ErrMsgPipelineDef
				defKind.Metadata.Labels = map[string]string{
					"branchName": pp.BranchName,
				}
				defKind.Metadata.Annotations = map[string]string{
					"envs":             strings.Join(pp.Envs, ","),
					"envProductions":   strings.Join(pp.EnvProductions, ","),
					"isDefault":        fmt.Sprintf("%v", pp.IsDefault),
					"webhookPushEvent": fmt.Sprintf("%v", pp.WebhookPushEvent),
					"tagSuffix":        pp.TagSuffix,
				}
				defKind.Items = append(defKind.Items, pp.PipelineDef)
				defKinds = append(defKinds, defKind)
			}
		}

		if len(project.ProjectDef.DockerIgnoreDefs) > 0 {
			defKind := defKindProject
			defKind.Kind = "dockerIgnoreDefs"
			for _, def := range project.ProjectDef.DockerIgnoreDefs {
				defKind.Items = append(defKind.Items, def)
			}
			defKinds = append(defKinds, defKind)
		}

		if len(project.ProjectDef.CustomOpsDefs) > 0 {
			defKind := defKindProject
			defKind.Kind = "customOpsDefs"
			defKind.Status.ErrMsg = project.ProjectDef.ErrMsgCustomOpsDefs
			for _, def := range project.ProjectDef.CustomOpsDefs {
				var isShow bool
				if len(o.ModuleNames) == 0 {
					isShow = true
				} else {
					for _, moduleName := range o.ModuleNames {
						if moduleName == def.CustomOpsName {
							isShow = true
							break
						}
					}
				}
				if isShow {
					defKind.Items = append(defKind.Items, def)
				}
			}
			defKinds = append(defKinds, defKind)
		}
	}

	defKindFilters := []pkg.DefKind{}
	if len(o.Param.Kinds) == 0 || o.Param.IsAllKind {
		defKindFilters = defKinds
	} else {
		for _, defKind := range defKinds {
			for _, kind := range o.Param.Kinds {
				if kind == defKind.Kind {
					defKindFilters = append(defKindFilters, defKind)
					break
				}
			}
		}
	}

	errMsgs := []string{}
	if project.ProjectDef.ErrMsgPackageDefs != "" {
		errMsg := fmt.Sprintf("packageDefs error: %s", project.ProjectDef.ErrMsgPackageDefs)
		errMsgs = append(errMsgs, errMsg)
	}
	for _, pae := range project.ProjectAvailableEnvs {
		if pae.ErrMsgDeployContainerDefs != "" {
			errMsg := fmt.Sprintf("deployContainerDefs envName=%s error: %s", pae.EnvName, pae.ErrMsgDeployContainerDefs)
			errMsgs = append(errMsgs, errMsg)
		}
	}
	for _, pae := range project.ProjectAvailableEnvs {
		for stepName, msg := range pae.ErrMsgCustomStepDefs {
			errMsg := fmt.Sprintf("customStepDef stepName=%s envName=%s error: %s", stepName, pae.EnvName, msg)
			errMsgs = append(errMsgs, errMsg)
		}
	}
	if len(project.ProjectDef.ErrMsgCustomStepDefs) > 0 {
		for stepName, msg := range project.ProjectDef.ErrMsgCustomStepDefs {
			errMsg := fmt.Sprintf("customStepDef stepName=%s error: %s", stepName, msg)
			errMsgs = append(errMsgs, errMsg)
		}
	}
	for _, pp := range project.ProjectPipelines {
		if pp.ErrMsgPipelineDef != "" {
			errMsg := fmt.Sprintf("pipelineDef branchName=%s error: %s", pp.BranchName, pp.ErrMsgPipelineDef)
			errMsgs = append(errMsgs, errMsg)
		}
	}

	if project.ProjectDef.ErrMsgCustomOpsDefs != "" {
		errMsg := fmt.Sprintf("customOpsDefs error: %s", project.ProjectDef.ErrMsgCustomOpsDefs)
		errMsgs = append(errMsgs, errMsg)
	}

	defKindList := pkg.DefKindList{
		Kind: "list",
		Defs: defKindFilters,
	}
	defKindList.Status.ErrMsgs = errMsgs

	dataOutput := map[string]interface{}{}
	m := map[string]interface{}{}
	bs, _ = json.Marshal(defKindList)
	_ = json.Unmarshal(bs, &m)
	if o.Full {
		dataOutput = m
	} else {
		dataOutput = pkg.RemoveMapEmptyItems(m)
	}

	switch o.Output {
	case "json":
		bs, _ = json.MarshalIndent(dataOutput, "", "  ")
		fmt.Println(string(bs))
	case "yaml":
		bs, _ = pkg.YamlIndent(dataOutput)
		fmt.Println(string(bs))
	default:
		for _, defKind := range defKindList.Defs {
			if defKind.Status.ErrMsg != "" {
				log.Error(defKind.Status.ErrMsg)
			}

			dataHeader := []string{}
			dataRows := [][]string{}
			bs, _ := json.Marshal(defKind.Items)
			switch defKind.Kind {
			case "projectSummary":
				items := []pkg.ProjectSummary{}
				_ = json.Unmarshal(bs, &items)
				for _, item := range items {
					var customSteps []string
					for _, conf := range item.CustomStepConfs {
						var isEnvDiff string
						if conf.IsEnvDiff {
							isEnvDiff = "[env]"
						}
						s := fmt.Sprintf("%s%s", conf.CustomStepName, isEnvDiff)
						customSteps = append(customSteps, s)
					}
					var nodePorts []string
					for _, port := range item.NodePorts {
						s := fmt.Sprintf("%d", port)
						nodePorts = append(nodePorts, s)
					}
					dataRow := []string{defKind.Kind, strings.Join(item.BuildNames, "\n"), strings.Join(item.PackageNames, "\n"), strings.Join(customSteps, "\n"), strings.Join(item.BranchNames, "\n"), strings.Join(item.EnvNames, "\n"), strings.Join(nodePorts, "\n")}
					dataRows = append(dataRows, dataRow)
				}
				dataHeader = []string{"kind", "Builds", "Packages", "CustomSteps", "Branches", "Envs", "NodePorts"}
			case "buildDefs":
				items := []pkg.BuildDef{}
				_ = json.Unmarshal(bs, &items)
				for _, item := range items {
					dataRow := []string{fmt.Sprintf("%s/%s", defKind.Kind, item.BuildName), item.BuildEnv, item.BuildPath, fmt.Sprintf("%d", item.BuildPhaseID), strings.Join(item.BuildCmds, "\n")}
					dataRows = append(dataRows, dataRow)
				}
				dataHeader = []string{"Name", "Env", "Path", "PhaseID", "Cmds"}
			case "packageDefs":
				items := []pkg.PackageDef{}
				_ = json.Unmarshal(bs, &items)
				for _, item := range items {
					dataRow := []string{fmt.Sprintf("%s/%s", defKind.Kind, item.PackageName), strings.Join(item.RelatedBuilds, "\n"), item.PackageFrom, strings.Join(item.Packages, "\n")}
					dataRows = append(dataRows, dataRow)
				}
				dataHeader = []string{"Name", "Builds", "From", "Dockerfile"}
			case "deployContainerDefs":
				items := []pkg.DeployContainerDef{}
				_ = json.Unmarshal(bs, &items)
				for _, item := range items {
					var ports []string
					for _, p := range item.DeployLocalPorts {
						if p.Protocol == "" {
							p.Protocol = "TCP"
						}
						ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
					}
					for _, p := range item.DeployNodePorts {
						if p.Protocol == "" {
							p.Protocol = "TCP"
						}
						ports = append(ports, fmt.Sprintf("%d:%d/%s", p.Port, p.NodePort, p.Protocol))
					}

					dependServices := []string{}
					for _, ds := range item.DependServices {
						dependServices = append(dependServices, fmt.Sprintf("%s:%d", ds.DependName, ds.DependPort))
					}
					dataRow := []string{fmt.Sprintf("%s/%s", defKind.Kind, item.DeployName), defKind.Metadata.Labels["envName"], item.RelatedPackage, fmt.Sprintf("%d", item.DeployReplicas), strings.Join(ports, ","), strings.Join(dependServices, "\n")}
					dataRows = append(dataRows, dataRow)
				}
				dataHeader = []string{"Name", "env", "Package", "Replicas", "Ports", "Depends"}
			case "customStepDef":
				items := []pkg.CustomStepModuleDef{}
				_ = json.Unmarshal(bs, &items)
				var envName string
				for k, v := range defKind.Metadata.Labels {
					if k == "envName" {
						envName = v
					}
				}
				for _, item := range items {
					dataRow := []string{fmt.Sprintf("%s/%s", defKind.Kind, item.ModuleName), defKind.Metadata.Labels["stepName"], envName, defKind.Metadata.Labels["enableMode"], strings.Join(item.RelatedStepModules, "\n"), fmt.Sprintf("%v", item.ManualEnable), item.ParamInputYaml}
					dataRows = append(dataRows, dataRow)
				}
				dataHeader = []string{"Name", "StepName", "Env", "EnableMode", "RelateModules", "ManualEnable", "Params"}
			case "pipelineDef":
				items := []pkg.PipelineDef{}
				_ = json.Unmarshal(bs, &items)
				for _, item := range items {
					var builds []string
					for _, build := range item.Builds {
						buildStr := fmt.Sprintf("%s: %v", build.Name, build.Run)
						builds = append(builds, buildStr)
					}
					envs := strings.Split(defKind.Metadata.Annotations["envs"], ",")
					envProductions := strings.Split(defKind.Metadata.Annotations["envProductions"], ",")
					dataRow := []string{fmt.Sprintf("%s/%s", defKind.Kind, defKind.Metadata.Labels["branchName"]), strings.Join(envs, "\n"), strings.Join(envProductions, "\n"), fmt.Sprintf("%v", item.IsAutoDetectBuild), fmt.Sprintf("%v", item.IsQueue), strings.Join(builds, "\n")}
					dataRows = append(dataRows, dataRow)
				}
				dataHeader = []string{"Name", "Envs", "EnvProds", "AutoDetect", "Queue", "Builds"}
			case "dockerIgnoreDefs":
				items := []string{}
				_ = json.Unmarshal(bs, &items)
				for _, item := range items {
					dataRow := []string{defKind.Kind, item}
					dataRows = append(dataRows, dataRow)
				}
				dataHeader = []string{"Name", "Value"}
			case "customOpsDefs":
				items := []pkg.CustomOpsDef{}
				_ = json.Unmarshal(bs, &items)
				for _, item := range items {
					dataRow := []string{fmt.Sprintf("%s/%s", defKind.Kind, item.CustomOpsName), item.CustomOpsDesc, strings.Join(item.CustomOpsSteps, "\n")}
					dataRows = append(dataRows, dataRow)
				}
				dataHeader = []string{"Name", "Desc", "Steps"}
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader(dataHeader)
			table.SetAutoWrapText(false)
			table.SetAutoFormatHeaders(true)
			table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetCenterSeparator("")
			table.SetColumnSeparator("")
			table.SetRowSeparator("")
			table.SetHeaderLine(false)
			table.SetBorder(false)
			table.SetTablePadding("\t")
			table.SetNoWhiteSpace(true)
			table.AppendBulk(dataRows)
			table.Render()
			fmt.Println()
		}

		if len(defKindList.Status.ErrMsgs) > 0 {
			fmt.Println("ERRORS")
			for _, errMsg := range defKindList.Status.ErrMsgs {
				log.Error(errMsg)
			}
			fmt.Println()
		}
	}

	return err
}
