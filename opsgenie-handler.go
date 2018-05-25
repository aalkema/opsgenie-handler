package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/sensu/sensu-go/types"
	"github.com/spf13/cobra"
)

var (
	apiKey string
	stdin  *os.File
)

func main() {
	rootCmd := configureRootCommand()
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err.Error())
	}
}

func configureRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "handler-opsgenie",
		Short: "an opsgenie handler built for use with sensu",
		RunE:  run,
	}

	cmd.Flags().StringVarP(&apiKey,
		"apiKey",
		"a",
		"",
		"The apiKey for the opsgenie integration")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		_ = cmd.Help()
		return errors.New("invalid argument(s) received")
	}
	if stdin == nil {
		stdin = os.Stdin
	}

	eventJSON, err := ioutil.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %s", err.Error())
	}

	event := &types.Event{}
	err = json.Unmarshal(eventJSON, event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal stdin data: %s", eventJSON)
	}

	if err = validateEvent(event); err != nil {
		return errors.New(err.Error())
	}

	if err = sendMessage(event); err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func sendMessage(event *types.Event) error {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(apiKey)
	alertCli, _ := cli.AlertV2()

	request := alertsv2.CreateAlertRequest{
		Message:     "SDL " + event.Entity.ID + " stopped",
		Alias:       event.Entity.ID + " stopped",
		Description: "SDL service on " + event.Entity.ID + " is stopped",
		Tags:        []string{"SDL"},
		Details: map[string]string{
			"check": event.Check.Name,
		},
		Entity:   event.Entity.ID,
		Source:   "Sensu",
		Priority: alertsv2.P3,
		User:     "user@opsgenie.com",
	}

	response, err := alertCli.Create(request)

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	fmt.Println("Create request ID: " + response.RequestID)
	return nil
}

func validateEvent(event *types.Event) error {
	if event.Timestamp <= 0 {
		return errors.New("timestamp is missing or must be greater than zero")
	}

	if event.Entity == nil {
		return errors.New("entity is missing from event")
	}

	if event.Check == nil {
		return errors.New("check is missing from event")
	}

	if err := event.Entity.Validate(); err != nil {
		return err
	}

	if err := event.Check.Validate(); err != nil {
		return errors.New(err.Error())
	}

	return nil
}
