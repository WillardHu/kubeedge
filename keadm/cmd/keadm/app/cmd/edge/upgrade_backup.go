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
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	edgeconfig "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/util/files"
	"github.com/kubeedge/kubeedge/pkg/version"
)

func NewBackupCommand() *cobra.Command {
	var opts BaseOptions
	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Back up important files for rollback edgecore.",
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
			// TODO: report result
			return backup(&opts, cfg)
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
	AddBaseFlags(cmd, &opts)
	return cmd
}

func backup(opts *BaseOptions, config *edgeconfig.EdgeCoreConfig) error {
	klog.Infof("backup process start ...")
	backupFiles := []string{
		config.DataBase.DataSource,
		opts.Config,
		filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName),
	}
	backupPath := filepath.Join(util.KubeEdgeBackupPath, version.Get().String())
	for _, file := range backupFiles {
		dest := filepath.Join(backupPath, filepath.Base(file))
		if err := files.FileCopy(file, dest); err != nil {
			return fmt.Errorf("failed to backup file %s, err: %v", file, err)
		}
	}
	klog.Infof("backup process successful")
	return nil
}
