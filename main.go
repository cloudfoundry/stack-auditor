package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
)

type Plugin struct{}

func main() {
	plugin.Start(new(Plugin))
}

func (c *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) == 0 {
		err := errors.New("no command line arguments provided")
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Ensure that we called the command audit-stack
	switch args[0] {
	case "audit-stack":
		info, err := Audit(cliConnection)
		if err != nil {
			log.Fatalf("error talking to cf: %v\n", err)
		}

		fmt.Println(info)

		exitChan := make(chan struct{})
		signalChan := make(chan os.Signal, 1)

		signal.Notify(make(chan os.Signal), syscall.SIGHUP)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-signalChan
			close(exitChan)
		}()

		timer := time.NewTimer(10 * time.Second)

		select {
		case <-timer.C:
			fmt.Println("10 seconds elapsed")
		case <-exitChan:
			os.Exit(128)
		}
	case "CLI-MESSAGE-UNINSTALL":
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "Unknown argument provided")
		os.Exit(17)
	}
}

func (c *Plugin) GetMetadata() plugin.PluginMetadata {
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
				Name:     "audit-stack",
				HelpText: "Audit stack command's help text",

				// UsageDetails is optional
				// It is used to show help of usage of each command
				UsageDetails: plugin.Usage{
					Usage: "audit-stack\n   cf audit-stack",
				},
			},
		},
	}
}

type GuidName map[string]string

func Audit(cliConnection plugin.CliConnection) (string, error) {
	orgs, err := cliConnection.GetOrgs()
	if err != nil {
		return "", err
	}

	orgMap := makeOrgMap(orgs)
	fmt.Println("orgMap: ", orgMap)

	spaceJSON, _ := cliConnection.CliCommandWithoutTerminalOutput("curl", "/v2/spaces")
	stackJSON, _ := cliConnection.CliCommandWithoutTerminalOutput("curl", "/v2/stacks")
	appJSON, _ := cliConnection.CliCommandWithoutTerminalOutput("curl", "/v2/apps")

	fmt.Printf("%v \n\n", appJSON[0])
	fmt.Println("-------------------------------------------------")
	fmt.Printf("%v \n\n", spaceJSON[0])
	fmt.Println("-------------------------------------------------")
	fmt.Printf("%v \n\n", stackJSON[0])
	fmt.Println("-------------------------------------------------")
	fmt.Printf("%d \n\n", len(spaceJSON))

	var g GuidName

	if err := json.Unmarshal([]byte(spaceJSON[0]), &g); err != nil {
		log.Fatal(err)
	}

	return "", nil
}

func makeOrgMap(orgs []plugin_models.GetOrgs_Model) map[string]string {
	m := make(map[string]string)

	for _, org := range orgs {
		m[org.Guid] = org.Name
	}
	return m
}

func (g *GuidName) UnmarshalJSON(b []byte) error {
	// if it is a space...
	return g.spaceUnmarshal(b)
}

func (g GuidName) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string(g))
}

func (g *GuidName) spaceUnmarshal(b []byte) error {
	var i map[string]interface{}
	keys := make([]string, 0, len(i))
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}
	for k := range i {
		keys = append(keys, k)
	}

	fmt.Println("Key: ", keys)
	return nil
}
