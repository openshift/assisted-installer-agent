package monitor

import (
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/openshift/baremetal-runtimecfg/pkg/config"
	"github.com/openshift/baremetal-runtimecfg/pkg/render"
	"github.com/openshift/baremetal-runtimecfg/pkg/utils"
	"github.com/sirupsen/logrus"
)

func DnsmasqWatch(kubeconfigPath, templatePath, cfgPath string, apiVip net.IP, interval time.Duration) error {
	signals := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	prevMD5 := ""

	signal.Notify(signals, syscall.SIGTERM)
	signal.Notify(signals, syscall.SIGINT)
	go func() {
		<-signals
		done <- true
	}()

	for {
		select {
		case <-done:
			return nil
		default:
			// We only care about the api vip and cluster domain here
			config, err := config.GetConfig(kubeconfigPath, "", "/etc/resolv.conf", apiVip, apiVip, 0, 0, 0)
			if err != nil {
				return err
			}
			tmpFile, err := ioutil.TempFile("", "")
			if err != nil {
				return err
			}
			defer os.Remove(tmpFile.Name())
			err = render.RenderFile(tmpFile.Name(), templatePath, config)
			if err != nil {
				log.WithFields(logrus.Fields{
					"config":  config,
					"tmpFile": tmpFile.Name(),
				}).Error("Failed to render dnsmasq host file")
				return err
			}
			newMD5, err := utils.GetFileMd5(tmpFile.Name())
			if err != nil {
				return err
			}
			log.WithFields(logrus.Fields{
				"prevMD5": prevMD5,
				"newMD5":  newMD5,
			}).Info("Md5s")
			if prevMD5 != newMD5 {
				err = render.RenderFile(cfgPath, templatePath, config)
				if err != nil {
					log.WithFields(logrus.Fields{
						"config":  config,
						"tmpFile": tmpFile.Name(),
					}).Error("Failed to render dnsmasq host file")
					return err
				}
				prevMD5 = newMD5
				err = ReloadDnsmasq()
				if err != nil {
					log.Error("Failed to reload dnsmasq configuration")
					return err
				}
				log.Info("Reloaded dnsmasq")
			}
			time.Sleep(interval)
		}
	}
}

func ReloadDnsmasq() error {
	cmd := exec.Command("dbus-send", "--system", "--dest=uk.org.thekelleys.dnsmasq", "/uk/org/thekelleys/dnsmasq", "uk.org.thekelleys.ClearCache")
	return cmd.Run()
}
