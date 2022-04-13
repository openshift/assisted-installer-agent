package monitor

import (
	"net"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/openshift/baremetal-runtimecfg/pkg/config"
	"github.com/openshift/baremetal-runtimecfg/pkg/render"
	"github.com/openshift/baremetal-runtimecfg/pkg/utils"
	"github.com/sirupsen/logrus"
)

const resolvConfFilepath string = "/var/run/NetworkManager/resolv.conf"

func CorednsWatch(kubeconfigPath, clusterConfigPath, templatePath, cfgPath string, apiVip, ingressVip net.IP, interval time.Duration) error {
	signals := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(signals, syscall.SIGTERM)
	signal.Notify(signals, syscall.SIGINT)
	go func() {
		<-signals
		done <- true
	}()

	prevMD5, err := utils.GetFileMd5(resolvConfFilepath)
	if err != nil {
		return err
	}
	prevConfig := config.Node{}

	for {
		select {
		case <-done:
			return nil
		default:
			curMD5, err := utils.GetFileMd5(resolvConfFilepath)
			if err != nil {
				return err
			}
			newConfig, err := config.GetConfig(kubeconfigPath, clusterConfigPath, resolvConfFilepath, apiVip, ingressVip, 0, 0, 0)
			if err != nil {
				return err
			}
			config.PopulateNodeAddresses(kubeconfigPath, &newConfig)
			// There should never be 0 nodes in a functioning cluster. This means
			// we failed to populate the list, so we don't want to render.
			if len(newConfig.Cluster.NodeAddresses) == 0 {
				time.Sleep(interval)
				continue
			}
			sort.SliceStable(newConfig.Cluster.NodeAddresses, func(i, j int) bool {
				return newConfig.Cluster.NodeAddresses[i].Name < newConfig.Cluster.NodeAddresses[j].Name
			})
			addressesChanged := len(newConfig.Cluster.NodeAddresses) != len(prevConfig.Cluster.NodeAddresses)
			if !addressesChanged {
				for i, addr := range newConfig.Cluster.NodeAddresses {
					if addr.Name != prevConfig.Cluster.NodeAddresses[i].Name {
						addressesChanged = true
						break
					}
				}
			}
			if curMD5 != prevMD5 || addressesChanged {
				if addressesChanged {
					log.WithFields(logrus.Fields{
						"Node Addresses": newConfig.Cluster.NodeAddresses,
					}).Info("Node change detected, rendering Corefile")
				} else {
					log.WithFields(logrus.Fields{
						"DNS upstreams": newConfig.DNSUpstreams,
					}).Info("Resolv.conf change detected, rendering Corefile")
				}
				err = render.RenderFile(cfgPath, templatePath, newConfig)
				if err != nil {
					log.WithFields(logrus.Fields{
						"config": newConfig,
					}).Error("Failed to render coredns Corefile")
					return err
				}
			}
			prevMD5 = curMD5
			prevConfig = newConfig
			time.Sleep(interval)
		}
	}
}
