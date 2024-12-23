//go:build !windows

/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package edge

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	phases "k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/reset"
	utilsexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/extsystem"
)

var (
	resetLongDescription = `
keadm reset edge command can be executed edge node
In edge node it shuts down the edge processes of KubeEdge
`
	resetExample = `
For edge node edge:
keadm reset edge
`
)

func NewOtherEdgeReset() *cobra.Command {
	reset := util.NewResetOptions()
	var cmd = &cobra.Command{
		Use:     "edge",
		Short:   "Teardowns EdgeCore component",
		Long:    resetLongDescription,
		Example: resetExample,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if reset.PreRun != "" {
				fmt.Printf("Executing pre-run script: %s\n", reset.PreRun)
				if err := util.RunScript(reset.PreRun); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if !reset.Force {
				fmt.Println("[reset] WARNING: Changes made to this host by 'keadm init' or 'keadm join' will be reverted.")
				fmt.Print("[reset] Are you sure you want to proceed? [y/N]: ")
				s := bufio.NewScanner(os.Stdin)
				s.Scan()
				if err := s.Err(); err != nil {
					return err
				}
				if strings.ToLower(s.Text()) != "y" {
					return fmt.Errorf("aborted reset operation")
				}
			}

			staticPodPath := ""
			config, err := util.ParseEdgecoreConfig(common.EdgecoreConfigPath)
			if err != nil {
				fmt.Printf("Failed to get edgecore's config with err:%v\n", err)
			} else {
				if reset.Endpoint == "" {
					reset.Endpoint = config.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint
				}
				staticPodPath = config.Modules.Edged.TailoredKubeletConfig.StaticPodPath
			}
			// first cleanup edge node static pod directory to stop static and mirror pod
			if staticPodPath != "" {
				if err := phases.CleanDir(staticPodPath); err != nil {
					fmt.Printf("Failed to delete static pod directory %s: %v\n", staticPodPath, err)
				} else {
					time.Sleep(1 * time.Second)
					fmt.Printf("Static pod directory has been removed!\n")
				}
			}

			// 1. kill edgecore process.
			// For edgecore, don't delete node from K8S
			if err := TearDownEdgeCore(); err != nil {
				return err
			}

			// 2. Remove containers managed by KubeEdge. Only for edge node.
			if err := util.RemoveContainers(reset.Endpoint, utilsexec.New()); err != nil {
				fmt.Printf("Failed to remove containers: %v\n", err)
			}

			// 3. Clean stateful directories
			if err := util.CleanDirectories(true); err != nil {
				return err
			}

			//4. TODO: clean status information

			return nil
		},

		PostRunE: func(_ *cobra.Command, _ []string) error {
			// post-run script
			if reset.PostRun != "" {
				fmt.Printf("Executing post-run script: %s\n", reset.PostRun)
				if err := util.RunScript(reset.PostRun); err != nil {
					fmt.Printf("Execute post-run script: %s failed: %v\n", reset.PostRun, err)
				}
			}
			return nil
		},
	}

	addResetFlags(cmd, reset)
	return cmd
}

// TearDownEdgeCore will bring down edge component,
func TearDownEdgeCore() error {
	extSystem, err := extsystem.GetExtSystem()
	if err != nil {
		return fmt.Errorf("failed to get init system, err: %v", err)
	}
	if extSystem.ServiceExists(util.KubeEdgeBinaryName) {
		if err := extSystem.ServiceStop(util.KubeEdgeBinaryName); err != nil {
			fmt.Printf("Failed to stop edgecore service, err: %v\n", err)
			return nil
		}
		if extSystem.ServiceIsEnabled(util.KubeEdgeBinaryName) {
			if err := extSystem.ServiceDisable(util.KubeEdgeBinaryName); err != nil {
				fmt.Printf("Failed to disable edgecore service, err: %v\n", err)
				return nil
			}
		}
		if err := extSystem.ServiceRemove(util.KubeEdgeBinaryName); err != nil {
			fmt.Printf("Failed to remove edgecore service, err: %v\n", err)
			return nil
		}
	}
	return nil
}

func addResetFlags(cmd *cobra.Command, resetOpts *common.ResetOptions) {
	cmd.Flags().BoolVar(&resetOpts.Force, "force", resetOpts.Force,
		"Reset the node without prompting for confirmation")
	cmd.Flags().StringVar(&resetOpts.Endpoint, "remote-runtime-endpoint", resetOpts.Endpoint,
		"Use this key to set container runtime endpoint")
	cmd.Flags().StringVar(&resetOpts.PreRun, common.FlagNamePreRun, resetOpts.PreRun,
		"Execute the prescript before resetting the node. (for example: keadm reset edge --pre-run=./test-script.sh ...)")
	cmd.Flags().StringVar(&resetOpts.PostRun, common.FlagNamePostRun, resetOpts.PostRun,
		"Execute the postscript after resetting the node. (for example: keadm reset edge --post-run=./test-script.sh ...)")
}
