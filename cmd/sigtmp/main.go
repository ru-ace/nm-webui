package main

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/godbus/dbus/v5"
)

func main() {
	conn, _ := dbus.SystemBus()
	defer conn.Close()
	ch := make(chan *dbus.Signal, 256)
	conn.Signal(ch)
	_ = conn.AddMatchSignal()

	go func() {
		time.Sleep(2 * time.Second)
		out, err := exec.Command("nmcli", "device", "wifi", "rescan").CombinedOutput()
		fmt.Println(">> nmcli rescan:", string(out), err)
	}()

	seen := map[string]int{}
	t := time.After(30 * time.Second)
	for {
		select {
		case sig := <-ch:
			seen[string(sig.Name)]++
			if sig.Name == "org.freedesktop.NetworkManager.Device.Wireless.ScanDone" {
				fmt.Printf("GOT ScanDone path=%s body=%v\n", sig.Path, sig.Body)
			}
		case <-t:
			fmt.Println("done, counts:")
			for k, v := range seen {
				fmt.Printf("  %d  %s\n", v, k)
			}
			return
		}
	}
}
