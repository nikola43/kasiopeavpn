//go:build linux
// +build linux

package main

import (
	"fmt"
	"log"
	"net"

	"github.com/fatih/color"
	"github.com/milosgajdos/tenus"
	"github.com/nikola43/kasiopeavpn/netlink"
	"github.com/songgao/water"
)

const (
	// MTU used for tunneled packets
	MTU = 1300
)

// ifaceSetup returns new interface OR PANIC!
func IfaceSetup(localCIDR string) *water.Interface {

	lIP, lNet, err := net.ParseCIDR(localCIDR)
	if nil != err {
		log.Fatalln("\nlocal ip is not in ip/cidr format")
		panic("invalid local ip")
	}

	iface, err := water.New(water.Config{DeviceType: water.TUN})

	if nil != err {
		log.Println("Unable to allocate TUN interface:", err)
		panic(err)
	}
	fmt.Println(color.CyanString("Interface allocated: "), color.YellowString(iface.Name()))

	link, err := tenus.NewLinkFrom(iface.Name())
	if nil != err {
		log.Fatalln("Unable to get interface info", err)
	}

	err = link.SetLinkMTU(MTU)
	if nil != err {
		log.Fatalln("Unable to set MTU to 1300 on interface")
	}

	err = link.SetLinkIp(lIP, lNet)
	if nil != err {
		log.Fatalln("Unable to set IP to ", lIP, "/", lNet, " on interface")
	}

	err = link.SetLinkUp()
	if nil != err {
		log.Fatalln("Unable to UP interface")
	}

	return iface
}

func RoutesThread(ifaceName string, refresh chan bool) {
	currentRoutes := map[string]bool{}
	for {
		<-refresh
		log.Println("Reloading routes...")
		conf := config.Load().(VPNState)

		routes2Del := map[string]bool{}

		for r := range currentRoutes {
			routes2Del[r] = true
		}

		for r := range conf.routes {
			rs := r.String()
			if _, exist := routes2Del[rs]; exist {
				delete(routes2Del, rs)
			} else {
				// real add route
				currentRoutes[rs] = true
				log.Println("Adding route:", rs)
				err := netlink.AddRoute(rs, "", "", ifaceName)
				if nil != err {
					log.Println("Adding route", rs, "failed:", err)
				}
			}
		}

		for r := range routes2Del {
			delete(currentRoutes, r)
			log.Println("Removing route:", r)
			err := netlink.DelRoute(r, "", "", ifaceName)
			if nil != err {
				log.Printf("Error removeing route \"%s\": %s", r, err.Error())
			}
		}
	}
}
