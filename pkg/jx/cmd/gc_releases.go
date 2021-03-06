package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
)

// GCReleasesOptions contains the CLI options for this command
type GCReleasesOptions struct {
	*CommonOptions

	RevisionHistoryLimit int
}

var (
	GCReleasesLong = templates.LongDesc(`
		Garbage collect the Jenkins X Activity Custom Resource Definitions

`)

	GCReleasesExample = templates.Examples(`
		jx garbage collect releases
		jx gc releases
`)
)

// NewCmd s a command object for the "step" command
func NewCmdGCReleases(commonOpts *CommonOptions) *cobra.Command {
	options := &GCReleasesOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "releases",
		Short:   "garbage collection for Releases",
		Long:    GCReleasesLong,
		Example: GCReleasesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().IntVarP(&options.RevisionHistoryLimit, "revision-history-limit", "l", 5, "Minimum number of Releases per application to keep")
	return cmd
}

// Run implements this command
func (o *GCReleasesOptions) Run() error {
	err := o.registerReleaseCRD()
	if err != nil {
		return err
	}

	client, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	// cannot use field selectors like `spec.kind=Preview` on CRDs so list all environments
	releaseInterface := client.JenkinsV1().Releases(ns)
	releases, err := releaseInterface.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(releases.Items) == 0 {
		// no preview environments found so lets return gracefully
		if o.Verbose {
			log.Info("no releases found\n")
		}
		return nil
	}

	jenkinsClient, err := o.JenkinsClient()
	if err != nil {
		return err
	}
	jobs, err := jenkinsClient.GetJobs()
	if err != nil {
		return err
	}
	var jobNames []string
	for _, j := range jobs {
		err = o.getAllPipelineJobNames(jenkinsClient, &jobNames, j.Name)
		if err != nil {
			return err
		}
	}

	pipelineReleases := make(map[string][]v1.Release)

	for _, a := range releases.Items {
		owner := a.Spec.GitOwner
		repo := a.Spec.GitRepository
		pipeline := owner + "/" + repo + "/master"
		// if activity has no job in Jenkins delete it
		matched := true
		if owner != "" && repo != "" {
			matched = false
			for _, j := range jobNames {
				if pipeline == j {
					matched = true
					break
				}
			}
		}
		if !matched {
			err = releaseInterface.Delete(a.Name, metav1.NewDeleteOptions(0))
			if err != nil {
				return err
			} else {
				log.Infof("Deleting Release %s as it no longer has a pipeline for %s\n", a.Name, pipeline)
			}
		}

		// collect all releases for a pipeline
		pipelineReleases[pipeline] = append(pipelineReleases[pipeline], a)
	}

	for _, releases := range pipelineReleases {
		kube.SortReleases(releases)

		// iterate over the old releases and remove them
		for i := o.RevisionHistoryLimit + 1; i < len(releases); i++ {
			name := releases[i].Name
			err = releaseInterface.Delete(name, metav1.NewDeleteOptions(0))
			if err != nil {
				return fmt.Errorf("failed to delete Release %s in namespace %s: %v\n", name, ns, err)
			} else {
				log.Infof("Deleting old Release %s\n", name)
			}
		}
	}
	return nil
}
