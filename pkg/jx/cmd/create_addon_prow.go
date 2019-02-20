package cmd

import (
	"github.com/spf13/cobra"

	"fmt"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

var (
	createAddonProwLong = templates.LongDesc(`
		Creates the Prow addon for handling webhook events
`)

	createAddonProwExample = templates.Examples(`
		# Create the Prow addon 
		jx create addon prow

		# Create the Prow addon in a custom namespace
		jx create addon prow -n mynamespace
	`)
)

const defaultProwVersion = ""

// CreateAddonProwOptions the options for the create spring command
type CreateAddonProwOptions struct {
	CreateAddonOptions
	Password        string
	Chart           string
	KnativePipeline bool
}

// NewCmdCreateAddonProw creates a command object for the "create" command
func NewCmdCreateAddonProw(commonOpts *CommonOptions) *cobra.Command {
	options := &CreateAddonProwOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "prow",
		Short:   "Create a Prow addon",
		Long:    createAddonProwLong,
		Example: createAddonProwExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addFlags(cmd, "", kube.DefaultProwReleaseName, defaultProwVersion)

	cmd.Flags().StringVarP(&options.Prow.Chart, optionChart, "c", kube.ChartProw, "The name of the chart to use")
	cmd.Flags().StringVarP(&options.Prow.HMACToken, "hmac-token", "", "", "OPTIONAL: The hmac-token is the token that you give to GitHub for validating webhooks. Generate it using any reasonable randomness-generator, eg openssl rand -hex 20")
	cmd.Flags().StringVarP(&options.Prow.OAUTHToken, "oauth-token", "", "", "OPTIONAL: The oauth-token is an OAuth2 token that has read and write access to the bot account. Generate it from the account's settings -> Personal access tokens -> Generate new token.")
	cmd.Flags().StringVarP(&options.Password, "password", "", "", "Overwrite the default admin password used to login to the Deck UI")
	cmd.Flags().BoolVarP(&options.KnativePipeline, "build-pipeline", "", true, "Enables Knative Build Pipeline. Otherwise we default to use Knative Build")
	return cmd
}

// Run implements the command
func (o *CreateAddonProwOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}

	err := o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that Helm is present")
	}
	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	o.Prow.Chart = o.Chart
	o.Prow.Version = o.Version
	o.Prow.SetValues = o.SetValues
	o.Namespace = o.currentNamespace

	isGitOps,_ := o.GetDevEnv()

	err = o.installProw(o.KnativePipeline, isGitOps, "", "")
	if err != nil {
		return fmt.Errorf("failed to install Prow: %v", err)
	}

	devNamespace, _, err := kube.GetDevNamespace(client, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	// create the ingress rule
	err = o.expose(devNamespace, devNamespace, o.Password)
	if err != nil {
		return err
	}

	return nil
}
