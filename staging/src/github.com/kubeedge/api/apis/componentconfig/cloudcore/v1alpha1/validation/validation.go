/*
Copyright 2019 The KubeEdge Authors.

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

package validation

import (
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	netutils "k8s.io/utils/net"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	utilvalidation "github.com/kubeedge/api/apis/util/validation"
)

// ValidateCloudCoreConfiguration validates `c` and returns an errorList if it is invalid
func ValidateCloudCoreConfiguration(c *v1alpha1.CloudCoreConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateKubeAPIConfig(*c.KubeAPIConfig)...)
	allErrs = append(allErrs, ValidateCommonConfig(*c.CommonConfig)...)
	allErrs = append(allErrs, ValidateModuleCloudHub(*c.Modules.CloudHub)...)
	allErrs = append(allErrs, ValidateModuleEdgeController(*c.Modules.EdgeController)...)
	allErrs = append(allErrs, ValidateModuleDeviceController(*c.Modules.DeviceController)...)
	allErrs = append(allErrs, ValidateModuleSyncController(*c.Modules.SyncController)...)
	allErrs = append(allErrs, ValidateModuleDynamicController(*c.Modules.DynamicController)...)
	allErrs = append(allErrs, ValidateModuleCloudStream(*c.Modules.CloudStream)...)
	return allErrs
}

func ValidateCommonConfig(c v1alpha1.CommonConfig) field.ErrorList {
	return validateHostPort(c.MonitorServer.BindAddress, field.NewPath("monitorServer.bindAddress"))
}

func validateHostPort(input string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	hostIP, port, err := net.SplitHostPort(input)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath, input, "must be IP:port"))
		return allErrs
	}

	if ip := netutils.ParseIPSloppy(hostIP); ip == nil {
		allErrs = append(allErrs, field.Invalid(fldPath, hostIP, "must be a valid IP"))
	}

	if p, err := strconv.Atoi(port); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath, port, "must be a valid port"))
	} else if p < 1 || p > 65535 {
		allErrs = append(allErrs, field.Invalid(fldPath, port, "must be a valid port"))
	}

	return allErrs
}

// ValidateModuleCloudHub validates `c` and returns an errorList if it is invalid
func ValidateModuleCloudHub(c v1alpha1.CloudHub) field.ErrorList {
	if !c.Enable {
		return field.ErrorList{}
	}

	allErrs := field.ErrorList{}
	validHTTPSPort := utilvalidation.IsValidPortNum(int(c.HTTPS.Port))
	validWPort := utilvalidation.IsValidPortNum(int(c.WebSocket.Port))
	validAddress := utilvalidation.IsValidIP(c.WebSocket.Address)
	validQPort := utilvalidation.IsValidPortNum(int(c.Quic.Port))
	validQAddress := utilvalidation.IsValidIP(c.Quic.Address)

	if len(validHTTPSPort) > 0 {
		for _, m := range validHTTPSPort {
			allErrs = append(allErrs, field.Invalid(field.NewPath("port"), c.HTTPS.Port, m))
		}
	}
	if len(validWPort) > 0 {
		for _, m := range validWPort {
			allErrs = append(allErrs, field.Invalid(field.NewPath("port"), c.WebSocket.Port, m))
		}
	}
	if len(validAddress) > 0 {
		for _, m := range validAddress {
			allErrs = append(allErrs, field.Invalid(field.NewPath("Address"), c.WebSocket.Address, m))
		}
	}
	if len(validQPort) > 0 {
		for _, m := range validQPort {
			allErrs = append(allErrs, field.Invalid(field.NewPath("port"), c.Quic.Port, m))
		}
	}
	if len(validQAddress) > 0 {
		for _, m := range validQAddress {
			allErrs = append(allErrs, field.Invalid(field.NewPath("Address"), c.Quic.Address, m))
		}
	}
	if !strings.HasPrefix(strings.ToLower(c.UnixSocket.Address), "unix://") {
		allErrs = append(allErrs, field.Invalid(field.NewPath("address"),
			c.UnixSocket.Address, "unixSocketAddress must has prefix unix://"))
	}
	s := strings.SplitN(c.UnixSocket.Address, "://", 2)
	if len(s) > 1 && !utilvalidation.FileIsExist(path.Dir(s[1])) {
		if err := os.MkdirAll(path.Dir(s[1]), os.ModePerm); err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("address"),
				c.UnixSocket.Address, fmt.Sprintf("create unixSocketAddress %v dir %v error: %v",
					c.UnixSocket.Address, path.Dir(s[1]), err)))
		}
	}
	if c.TokenRefreshDuration <= 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("TokenRefreshDuration"),
			c.TokenRefreshDuration, "TokenRefreshDuration must be positive"))
	}
	return allErrs
}

// ValidateModuleEdgeController validates `e` and returns an errorList if it is invalid
func ValidateModuleEdgeController(e v1alpha1.EdgeController) field.ErrorList {
	if !e.Enable {
		return field.ErrorList{}
	}
	allErrs := field.ErrorList{}
	if e.NodeUpdateFrequency <= 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("NodeUpdateFrequency"), e.NodeUpdateFrequency, "NodeUpdateFrequency need > 0"))
	}
	return allErrs
}

// ValidateModuleDeviceController validates `d` and returns an errorList if it is invalid
func ValidateModuleDeviceController(d v1alpha1.DeviceController) field.ErrorList {
	if !d.Enable {
		return field.ErrorList{}
	}

	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateModuleSyncController validates `d` and returns an errorList if it is invalid
func ValidateModuleSyncController(d v1alpha1.SyncController) field.ErrorList {
	if !d.Enable {
		return field.ErrorList{}
	}

	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateModuleDynamicController validates `d` and returns an errorList if it is invalid
func ValidateModuleDynamicController(d v1alpha1.DynamicController) field.ErrorList {
	if !d.Enable {
		return field.ErrorList{}
	}

	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateModuleCloudStream validates `d` and returns an errorList if it is invalid
func ValidateModuleCloudStream(d v1alpha1.CloudStream) field.ErrorList {
	if !d.Enable {
		return field.ErrorList{}
	}

	allErrs := field.ErrorList{}

	if !utilvalidation.FileIsExist(d.TLSTunnelPrivateKeyFile) {
		klog.Warningf("TLSTunnelPrivateKeyFile does not exist in %s, will load from secret", d.TLSTunnelPrivateKeyFile)
	}
	if !utilvalidation.FileIsExist(d.TLSTunnelCertFile) {
		klog.Warningf("TLSTunnelCertFile does not exist in %s, will load from secret", d.TLSTunnelCertFile)
	}
	if !utilvalidation.FileIsExist(d.TLSTunnelCAFile) {
		klog.Warningf("TLSTunnelCAFile does not exist in %s, will load from secret", d.TLSTunnelCAFile)
	}

	if !utilvalidation.FileIsExist(d.TLSStreamPrivateKeyFile) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("TLSStreamPrivateKeyFile"), d.TLSStreamPrivateKeyFile, "TLSStreamPrivateKeyFile not exist"))
	}
	if !utilvalidation.FileIsExist(d.TLSStreamCertFile) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("TLSStreamCertFile"), d.TLSStreamCertFile, "TLSStreamCertFile not exist"))
	}
	if !utilvalidation.FileIsExist(d.TLSStreamCAFile) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("TLSStreamCAFile"), d.TLSStreamCAFile, "TLSStreamCAFile not exist"))
	}

	return allErrs
}

// ValidateKubeAPIConfig validates `k` and returns an errorList if it is invalid
func ValidateKubeAPIConfig(k v1alpha1.KubeAPIConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	if k.KubeConfig != "" && !filepath.IsAbs(k.KubeConfig) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("kubeconfig"), k.KubeConfig, "kubeconfig need abs path"))
	}
	if k.KubeConfig != "" && !utilvalidation.FileIsExist(k.KubeConfig) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("kubeconfig"), k.KubeConfig, "kubeconfig not exist"))
	}
	return allErrs
}
