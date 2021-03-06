package cmd

import (
	"github.com/spf13/cobra"
)

// StepBuildPackOptions contains the command line flags
type StepBuildPackOptions struct {
	StepOptions
}

// NewCmdStepBuildPack Steps a command object for the "step" command
func NewCmdStepBuildPack(commonOpts *CommonOptions) *cobra.Command {
	options := &StepBuildPackOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "buildpack",
		Short:   "buildpack [command]",
		Aliases: buildPacksAliases,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdStepBuildPackApply(commonOpts))
	return cmd
}

// Run implements this command
func (o *StepBuildPackOptions) Run() error {
	return o.Cmd.Help()
}
