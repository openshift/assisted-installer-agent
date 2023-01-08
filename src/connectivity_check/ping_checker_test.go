package connectivity_check

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/assisted-service/models"
)

var _ = Describe("parse ping command", func() {

	tests := []struct {
		name         string
		cmdOutput    string
		averageRTTMs float64
		packetLoss   float64
		errFunc      func(string) string
	}{
		{name: "Nominal: no packet loss",
			cmdOutput: `PING www.acme.com (127.0.0.1) 56(84) bytes of data.

		--- www.acme.com ping statistics ---
		10 packets transmitted, 10 received, 0% packet loss, time 9011ms
		rtt min/avg/max/mdev = 14.278/17.099/19.136/1.876 ms`,
			averageRTTMs: 17.099,
			packetLoss:   0,
		},
		{name: "Nominal: with packet loss",
			cmdOutput: `PING 192.168.1.1 (192.168.1.1) 56(84) bytes of data.

		--- 192.168.1.1 ping statistics ---
		10 packets transmitted, 4 received, 60% packet loss, time 9164ms
		rtt min/avg/max/mdev = 2.616/2.871/3.183/0.255 ms`,
			averageRTTMs: 2.871,
			packetLoss:   60,
		},
		{name: "Nominal: with packet loss with decimals",
			cmdOutput: `PING 192.168.1.1 (192.168.1.1) 56(84) bytes of data.

		--- 192.168.1.1 ping statistics ---
		10 packets transmitted, 4 received, 23.33% packet loss, time 9164ms
		rtt min/avg/max/mdev = 2.616/2.871/3.183/0.255 ms`,
			averageRTTMs: 2.871,
			packetLoss:   23.33,
		},
		{name: "KO: unable to parse average RTT",
			cmdOutput: `PING 192.168.1.1 (192.168.1.1) 56(84) bytes of data.

			--- 192.168.1.1 ping statistics ---
			10 packets transmitted, 4 received, 60% packet loss, time 9164ms
			rtt min/average/max/mdev = 2.616/2.871/3.183/0.255 ms`,
			averageRTTMs: 0,
			packetLoss:   60,
			errFunc: func(s string) string {
				return fmt.Sprintf(`Unable to retrieve the average RTT for ping: unable to parse %s with regex rtt min\/avg\/max\/mdev = .*\/([^\/]+)\/.*\/.* ms`, s)
			},
		},
		{name: "KO: unable to parse packets loss percentage",
			cmdOutput: `PING 192.168.1.1 (192.168.1.1) 56(84) bytes of data.

			--- 192.168.1.1 ping statistics ---
			10 packets transmitted, 4 received, 60%  packet loss, time 9164ms
			rtt min/avg/max/mdev = 2.616/2.871/3.183/0.255 ms`,
			errFunc: func(s string) string {
				return fmt.Sprintf(`Unable to retrieve packet loss percentage: unable to parse %s with regex [\d]+ packets transmitted, [\d]+ received, (([\d]*[.])?[\d]+)%% packet loss, time [\d]+ms`, s)
			},
		},
	}
	for i := range tests {
		t := tests[i]
		It(t.name, func() {
			conn := models.L3Connectivity{}
			err := parsePingCmd(&conn, t.cmdOutput)
			if t.errFunc != nil {
				Expect(err.Error()).To(BeEquivalentTo(t.errFunc(t.cmdOutput)))
			} else {
				Expect(err).To(BeNil())
			}
			Expect(conn.AverageRTTMs).Should(Equal(t.averageRTTMs))
			Expect(conn.PacketLossPercentage).Should(Equal(t.packetLoss))
		})
	}
})
