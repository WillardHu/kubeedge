package edge

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	edgeconfig "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

func NewRollbackCommand() *cobra.Command {
	var opts RollbackOptions
	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Roll back the edge node to the desired version.",
		PreRunE: func(_cmd *cobra.Command, _args []string) error {
			// TODO: check HistoricalVersion and set default value if empty
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
			return rollback(&opts, cfg)
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
	AddRollbackFlags(cmd, &opts)
	return cmd
}

func rollback(opts *RollbackOptions, config *edgeconfig.EdgeCoreConfig) error {
	klog.Infof("rollback process start ...")
	// Stop origin edgecore.
	if err := util.KillKubeEdgeBinary(util.KubeEdgeBinaryName); err != nil {
		return fmt.Errorf("failed to stop edgecore, err: %v", err)
	}
	rollbackFilesPathMap := map[string]string{
		"edgecore.db":           config.DataBase.DataSource,
		"edgecore.yaml":         opts.Config,
		util.KubeEdgeBinaryName: filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName),
	}
	// Rollback backup files.
	backupPath := filepath.Join(util.KubeEdgeBackupPath, opts.HistoricalVersion)
	for backupFile, dest := range rollbackFilesPathMap {
		if err := files.FileCopy(filepath.Join(backupPath, backupFile), dest); err != nil {
			return fmt.Errorf("failed to rollback file %s, err: %v", dest, err)
		}
	}
	// Start new edgecore.
	if err := runEdgeCore(false); err != nil {
		return fmt.Errorf("failed to start edgecore, err: %v", err)
	}
	klog.Infof("rollback process successful")
	return nil
}
