package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/cloudfoundry/stack-auditor/audit"
	"github.com/cloudfoundry/stack-auditor/resources"
)

type StackAuditor struct{}

const (
	AuditStackCmd  = "audit-stack"
	ChangeStackCmd = "change-stack"
	ChangeStackSuccessMsg = "Application %s was successfully changed to Stack %s"
)

func main() {
	plugin.Start(new(StackAuditor))
}

func (s *StackAuditor) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) == 0 {
		err := errors.New("no command line arguments provided")
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Ensure that we called the command audit-stack
	switch args[0] {
	case AuditStackCmd:
		info, err := Audit(cliConnection)
		if err != nil {
			log.Fatalf("error talking to cf: %v\n", err)
		}
		fmt.Println(info)

	case ChangeStackCmd:
		if len(args) != 3 {
			log.Fatal("Incorrect number of arguments provided - Usage: cf change-stack <org/space/app> <stack>\n")
		}
		info, err := ChangeStack(cliConnection, args[1], args[2])
		if err != nil {
			log.Fatalf("error talking to cf: %v\n", err)
		}
		fmt.Println(info)

	case "CLI-MESSAGE-UNINSTALL":
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "Unknown argument provided")
		os.Exit(17)
	}
}

func (s *StackAuditor) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "StackAuditor",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 7,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     AuditStackCmd,
				HelpText: "Audit stack command's help text",

				UsageDetails: plugin.Usage{
					Usage: AuditStackCmd + "\n   cf audit-stack",
				},
			},
			{
				Name:     ChangeStackCmd,
				HelpText: "Change stack command's help text",

				UsageDetails: plugin.Usage{
					Usage: ChangeStackCmd + "\n   cf change-stack /org/space/app stack",
				},
			},
		},
	}
}

func Audit(cli plugin.CliConnection) (string, error) {
	a := audit.Actor{
		Conn: cli,
	}

	list, err := assembleEntries(a)
	if err != nil {
		return "", err
	}

	sort.Strings(list)

	return strings.Join(list, "\n") + "\n", nil
}

func ChangeStack(cli plugin.CliConnection, appName string, stackName string) (string, error) {
	a := audit.Actor{
		Conn: cli,
	}

	stackGuid, err := getStackGUID(a, stackName)
	if err != nil {
		return "", err
	}

	appGuid, err := getAppGUID(a, appName, stackName)
	if err != nil {
		return "", err
	}

	if _, err = cli.CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+appGuid, "-X", "PUT", `-d={"stack_guid":"`+stackGuid+`","state":"STOPPED"}`) ; err != nil {
		return "", err
	}

	if _, err = cli.CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+appGuid, "-X", "PUT", `-d={"state":"STARTED"}`); err != nil {
		return "", err
	}

	result := fmt.Sprintf(ChangeStackSuccessMsg, appName, stackName)
	return result, nil
}

func assembleEntries(a audit.Actor) ([]string, error) {
	var entries []string

	orgMap, spaceNameMap, spaceOrgMap, stackMap, allApps, err := getCFContext(a)
	if err != nil {
		return nil, err
	}


	for _, apps := range allApps {
		for _, app := range apps.Resources {
			appName := app.Entity.Name
			spaceName := spaceNameMap[app.Entity.SpaceGUID]
			stackName := stackMap[app.Entity.StackGUID]

			orgName := orgMap[spaceOrgMap[app.Entity.SpaceGUID]]
			entries = append(entries, fmt.Sprintf("%s/%s/%s %s", orgName, spaceName, appName, stackName))
		}
	}
	return entries, nil
}

func getCFContext(a audit.Actor) (orgMap, spaceNameMap, spaceOrgMap, stackMap map[string]string, allApps resources.Apps, err error){
	orgs, err := a.GetOrgs()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	allSpaces, err := a.GetAllSpaces()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	allStacks, err := a.GetAllStacks()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	allApps, err = a.GetAllApps()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	orgMap = orgs.Map()
	spaceNameMap, spaceOrgMap = allSpaces.MakeSpaceOrgAndNameMap()
	stackMap = allStacks.MakeStackMap()

	return orgMap, spaceNameMap, spaceOrgMap, stackMap, allApps, nil
}

func getStackGUID(a audit.Actor, stackName string) (string, error){
	allStacks, err := a.GetAllStacks()
	if err != nil {
		return "", err
	}

	stackGuid := ""
	stackMap := allStacks.MakeStackMap()
	for guid, val := range stackMap {
		if val == stackName {
			stackGuid = guid
			break
		}
	}
	if stackGuid == "" {
		return "", fmt.Errorf("%s is not a valid stack", stackName)
	}

	return stackGuid, nil
}

func getAppGUID(a audit.Actor, appString string, stackName string) (string, error){
	orgMap, spaceNameMap, spaceOrgMap, stackMap, allApps, err := getCFContext(a)
	if err != nil {
		return "", err
	}

	parts := strings.Split(appString, "/")
	if len(parts) != 3 {
		return "", errors.New("invalid App Name – doesn't have Org/Space/App")
	}
	orgName := parts[0]
	spaceName := parts[1]
	appName := parts[2]

	for _, apps := range allApps {
		for _, cur := range apps.Resources {
			curApp := cur.Entity.Name
			curSpace := spaceNameMap[cur.Entity.SpaceGUID]
			curStack := stackMap[cur.Entity.StackGUID]

			curOrg := orgMap[spaceOrgMap[cur.Entity.SpaceGUID]]

			if curOrg == orgName && curSpace == spaceName && curApp == appName {
				if curStack != stackName {
					return cur.Metadata.GUID, nil
				} else {
					return "", fmt.Errorf("application is already associated with stack %s", stackName)
				}
			}
		}
	}

	return "", errors.New("application could not be found")
}
