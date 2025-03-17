/*
Copyright 2025 The KubeEdge Authors.

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
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	edgeconfig "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

func NewUpgradeCommand() *cobra.Command {
	var opts UpgradeOptions
	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Upgrade the edge node to the desired version.",
		PreRunE: func(_cmd *cobra.Command, _args []string) error {
			if opts.PreRun != "" {
				fmt.Printf("Executing pre-run script: %s\n", opts.PreRun)
				if err := util.RunScript(opts.PreRun); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(_cmd *cobra.Command, _args []string) error {
			cfg, err := util.ParseEdgecoreConfig(opts.Config)
			if err != nil {
				return fmt.Errorf("failed to parse the edgecore config file %s, err: %v",
					opts.Config, err)
			}
			// TODO: auto rollback and report result
			return Upgrade(&opts, cfg)
		},
		PostRunE: func(_cmd *cobra.Command, _args []string) error {
			// post-run script
			if opts.PostRun != "" {
				fmt.Printf("Executing post-run script: %s\n", opts.PostRun)
				if err := util.RunScript(opts.PostRun); err != nil {
					fmt.Printf("Execute post-run script: %s failed: %v\n", opts.PostRun, err)
				}
			}
			return nil
		},
	}
	AddUpgradeFlags(cmd, &opts)
	return cmd
}

// Upgrade runs edge node upgrade
func Upgrade(opts *UpgradeOptions, config *edgeconfig.EdgeCoreConfig) error {
	ctx := context.Background()
	// Get new edgecore binary from the image.
	klog.Infof("begin to download %s of edgecore", opts.ToVersion)
	edgecorePath, err := getEdgeCoreBinary(ctx, opts, config)
	if err != nil {
		return fmt.Errorf("failed to get edgecore binary, err: %v", err)
	}
	klog.Infof("upgrade process start ...")
	// Stop origin edgecore.
	if err := util.KillKubeEdgeBinary(util.KubeEdgeBinaryName); err != nil {
		return fmt.Errorf("failed to stop edgecore, err: %v", err)
	}
	// Copy new edgecore to /usr/local/bin.
	dest := filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName)
	if err := files.FileCopy(edgecorePath, dest); err != nil {
		return fmt.Errorf("failed to copy edgecore to %s, err: %v", dest, err)
	}
	// Start new edgecore.
	if err := runEdgeCore(false); err != nil {
		return fmt.Errorf("failed to start edgecore, err: %v", err)
	}
	klog.Info("upgrade process successful")
	return nil
}

// getEdgeCoreBinary pulls the installation-package image and obtains the edgecore binary from it.
// The edgecore binary is copied to the upgrade path, and the filepath is returned.
func getEdgeCoreBinary(ctx context.Context, opts *UpgradeOptions, config *edgeconfig.EdgeCoreConfig,
) (string, error) {
	container, err := util.NewContainerRuntime(
		config.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint,
		config.Modules.Edged.TailoredKubeletConfig.CgroupDriver)
	if err != nil {
		return "", fmt.Errorf("failed to new container runtime, err: %v", err)
	}
	image := opts.Image + ":" + opts.ToVersion
	if err := container.PullImage(ctx, image, nil, nil); err != nil {
		return "", fmt.Errorf("failed to pull image %s, err: %v", image, err)
	}
	containerFilePath := filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName)
	hostPath := filepath.Join(upgradePath(opts.ToVersion), util.KubeEdgeBinaryName)
	files := map[string]string{containerFilePath: hostPath}
	if err := container.CopyResources(ctx, image, files); err != nil {
		return "", fmt.Errorf("failed to copy edgecore from %s in the image %s to host %s, err: %v",
			containerFilePath, image, hostPath, err)
	}
	return hostPath, nil
}

// upgradePath returns the path of the upgrade directory.
func upgradePath(ver string) string {
	return filepath.Join(util.KubeEdgeUpgradePath, ver)
}
