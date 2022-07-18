package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/fatih/color"
	encryption "github.com/nikola43/kasiopeavpn/encryption"
	packet "github.com/nikola43/kasiopeavpn/packet"
	"github.com/nikola43/kasiopeavpn/reuseport"
	"github.com/songgao/water"
	"golang.org/x/net/ipv4"
)

const (
	// AppVersion contains current application version for -version command flag
	AppVersion = "0.2.0b"
)

const (
	// I use TUN interface, so only plain IP packet,
	// no ethernet header + mtu is set to 1300

	// BUFFERSIZE is size of buffer to receive packets
	// (little bit bigger than maximum)
	BUFFERSIZE = 1518
)

func rcvrThread(proto string, port int, iface *water.Interface) {
	conn, err := reuseport.NewReusableUDPPortConn(proto, fmt.Sprintf(":%v", port))
	if nil != err {
		fmt.Println(color.RedString("Unable to get UDP socket"))
		fmt.Println(color.RedString(err.Error()))
		os.Exit(-1)
	}

	encrypted := make([]byte, BUFFERSIZE)
	var decrypted packet.IPPacket = make([]byte, BUFFERSIZE)

	for {
		n, _, err := conn.ReadFrom(encrypted)
		if err != nil {
			fmt.Println(color.RedString("Unable to get UDP socket"))
			fmt.Println(color.RedString(err.Error()))
			continue
		}

		// ReadFromUDP can return 0 bytes on timeout
		if 0 == n {
			continue
		}

		conf := config.Load().(VPNState)

		if !conf.Main.main.CheckSize(n) {
			fmt.Println(color.RedString("invalid packet size ", n))
			continue
		}

		size, mainErr := encryption.DecryptV4Chk(conf.Main.main, encrypted[:n], decrypted)
		if nil != mainErr {
			fmt.Println(color.RedString("mainErr ", mainErr))
			if nil != conf.Main.alt {
				size, err = encryption.DecryptV4Chk(conf.Main.alt, encrypted[:n], decrypted)
				if nil != err {
					fmt.Println(color.RedString("Corrupted package ", mainErr))
					log.Println("Corrupted package: ", mainErr, " / ", err)
					continue
				}
			} else {
				fmt.Println(color.RedString("Corrupted package ", mainErr))
				continue
			}
		}

		n, err = iface.Write(decrypted[:size])
		if nil != err {
			fmt.Println(color.RedString("Error writing to local interface ", err))
		} else if n != size {
			fmt.Println(color.RedString("Partial package written to local interface ", err))
		}
	}
}

func sndrThread(conn *net.UDPConn, iface *water.Interface) {
	// first time fill with random numbers
	ivbuf := make([]byte, config.Load().(VPNState).Main.main.IVLen())
	if _, err := io.ReadFull(rand.Reader, ivbuf); err != nil {
		fmt.Println(color.RedString("Unable to get rand data ", err))
	}

	var packet packet.IPPacket = make([]byte, BUFFERSIZE)
	var encrypted = make([]byte, BUFFERSIZE)

	for {
		plen, err := iface.Read(packet[:MTU])
		if err != nil {
			fmt.Println(color.RedString("err ", err))
			break
		}

		if 4 != packet.IPver() {
			header, _ := ipv4.ParseHeader(packet)
			//log.Printf("Non IPv4 packet [%+v]\n", header)
			fmt.Println(color.YellowString("Non IPv4 packet ", header))
			continue
		}

		// each time get pointer to (probably) new config
		c := config.Load().(VPNState)

		dst := packet.Dst()

		wanted := false

		addr, ok := c.remotes[dst]

		if ok {
			wanted = true
		}

		if dst == c.Main.bcastIP || packet.IsMulticast() {
			wanted = true
		}

		// very ugly and useful only for a limited numbers of routes!
		log.Println("wanted DstV4", wanted)
		var ip net.IP
		if !wanted {
			ip = packet.DstV4()
			log.Println("ip", ip)
			for n, s := range c.routes {
				if n.Contains(ip) {
					addr = s
					ok = true
					wanted = true
					break
				}
			}
		}
		log.Println("wanted", wanted)
		if wanted {
			log.Println("ip wanted", ip)
			// new len contatins also 2byte original size
			clen := c.Main.main.AdjustInputSize(plen)

			if clen+c.Main.main.OutputAdd() > len(packet) {
				log.Println("clen + data > len(package)", clen, len(packet))
				continue
			}

			tsize := c.Main.main.Encrypt(packet[:clen], encrypted, ivbuf)

			if ok {
				n, err := conn.WriteToUDP(encrypted[:tsize], addr)
				if nil != err {
					log.Println("Error sending package:", err)
				}
				if n != tsize {
					log.Println("Only ", n, " bytes of ", tsize, " sent")
				}
				log.Println("n WriteToUDP", n)
			} else {
				// multicast or broadcast
				for _, addr := range c.remotes {
					n, err := conn.WriteToUDP(encrypted[:tsize], addr)
					if nil != err {
						log.Println("Error sending package:", err)
					}
					if n != tsize {
						log.Println("Only ", n, " bytes of ", tsize, " sent")
					}
					log.Println("n broadcast", n)
				}
			}
		} else {
			log.Println("Unknown dst: ", dst)
		}
	}

}

func main() {

	// system config
	numCpu := runtime.NumCPU()
	usedCpu := numCpu
	runtime.GOMAXPROCS(usedCpu)

	PrintSystemInfo(numCpu, usedCpu)
	PrintNetworkStatus()
	PrintUserBalance("0xFABB0ac9d68B0B445fB7357272Ff202C5651694a", 932)
	PrintUserBalance2("0xFABB0ac9d68B0B445fB7357272Ff202C5651694a", 923)

	version := flag.Bool("version", false, "print lcvpn version")
	flag.Parse()
	if *version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	routeReload := make(chan bool, 1)

	InitConfig(routeReload)

	conf := config.Load().(VPNState)

	iface := IfaceSetup(conf.Main.local)

	// start routes changes in config monitoring
	go RoutesThread(iface.Name(), routeReload)

	log.Println("Interface parameters configured")

	// Start listen threads
	for i := 0; i < conf.Main.RecvThreads; i++ {
		go rcvrThread("udp4", conf.Main.Port, iface)
	}

	// init udp socket for write
	writeAddr, err := net.ResolveUDPAddr("udp", ":")
	if nil != err {
		log.Fatalln("Unable to get UDP socket:", err)
	}

	writeConn, err := net.ListenUDP("udp", writeAddr)
	if nil != err {
		log.Fatalln("Unable to create UDP socket:", err)
	}

	// Start sender threads
	for i := 0; i < conf.Main.SendThreads; i++ {
		go sndrThread(writeConn, iface)
	}
	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGTERM)

	<-exitChan

	err = writeConn.Close()
	if nil != err {
		log.Println("Error closing UDP connection: ", err)
	}
}

func PrintSystemInfo(numCpu, usedCpu int) {
	fmt.Println("")
	fmt.Println(color.YellowString("  ----------------- System Info -----------------"))
	fmt.Println(color.CyanString("\t    Number CPU cores available: "), color.GreenString(strconv.Itoa(numCpu)))
	fmt.Println(color.CyanString("\t    Used of CPU cores: "), color.YellowString(strconv.Itoa(usedCpu)))
	fmt.Println()
}

func PrintNetworkStatus() {
	fmt.Println(color.YellowString("  ----------------- Network Info -----------------"))
	fmt.Println(color.CyanString("\t    Number Nodes: "), color.YellowString(strconv.Itoa(3)))
	fmt.Println(color.CyanString("\t    Prague: "), color.YellowString(strconv.Itoa(1)))
	fmt.Println(color.CyanString("\t    Kiev: "), color.YellowString(strconv.Itoa(1)))
	fmt.Println(color.CyanString("\t    Singapour: "), color.YellowString(strconv.Itoa(1)))
	fmt.Println()
}

func PrintUserBalance(address string, balance int) {
	fmt.Println(color.YellowString("  ----------------- Node Owner -----------------"))
	fmt.Println(color.CyanString("  "), color.GreenString(address))
	fmt.Println(color.CyanString("\t    Balance: "), color.YellowString(strconv.Itoa(balance)), color.YellowString(" $ZOE"))
	fmt.Println()
}

func PrintUserBalance2(address string, balance int) {
	fmt.Println(color.YellowString("  ----------------- Network Info -----------------"))
	fmt.Println(color.CyanString("\t    Send: "), color.YellowString(strconv.Itoa(1732)), color.YellowString("MB"))
	fmt.Println(color.CyanString("\t    Received: "), color.YellowString(strconv.Itoa(1343)), color.YellowString("MB"))
	fmt.Println(color.CyanString("\t    Duration: "), color.YellowString("19:20:04"))
	fmt.Println(color.RedString("\t    Paid: "), color.YellowString(strconv.Itoa(1)), color.YellowString("$ZOE = 2.52$"))
	fmt.Println()
}
