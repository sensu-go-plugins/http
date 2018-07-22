package plugin

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Command is an alias of a Cobra command, used for abstraction of the cobra pkg
type Command = *cobra.Command

// RunFunc represents the signature for the run function of a plugin
type RunFunc = func(Command, []string) error

// Plugin represents a generic Sensu plugin
type Plugin interface {
	Command() Command
	Run() error
}

// NewCommand creates a new generic Sensu plugin CLI with the provided name
func NewCommand(name string) Command {
	return &cobra.Command{
		Use:           name,
		Short:         fmt.Sprintf("%s is a Sensu plugin", name),
		SilenceErrors: true,
		SilenceUsage:  true,
	}
}

// Execute executes the plugin command
func Execute(plugin Plugin) {
	cmd := plugin.Command()
	cmd.RunE = func(cmd Command, args []string) error {
		return plugin.Run()
	}

	// Execute the plugin
	if exit := cmd.Execute(); exit != nil {
		if exit, ok := exit.(*Exit); ok {
			fmt.Printf("%s %s", cmd.Name(), exit.Error())
			os.Exit(exit.Status)
		}

		//  Unknown exit code
		fmt.Printf("%s %s", cmd.Name(), exit.Error())
		os.Exit(4)
	}

	// No exit code was explicitly returned
	fmt.Printf("%s %s: %+v\n", cmd.Name(), Statuses[3], "check did not returned an exit code")
	os.Exit(4)
}
