package main

import (
	"context"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/gianarb/planner"
	"github.com/peak/go-config"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
	"time"
)

const (
	configFile = "/etc/edged/tailscale-prefs.yaml"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	configChan, err := config.Watch(ctx, configFile)
	if err != nil {
		panic(err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		prefs, _ := tailscale.GetPrefs(ctx)
		tailscalePlan := &TailscalePlan{
			TargetPrefs: prefs,
		}
		scheduler := planner.NewScheduler()
		logger := initLogger()
		scheduler.WithLogger(logger)
		for {
			select {
			case s := <-signalChan:
				switch s {
				case syscall.SIGINT, syscall.SIGTERM:
					logger.Fatal("got SIGINT/SIGTERM, exiting.")
					cancel()
					os.Exit(0)
				case syscall.SIGHUP:
					logger.Info("got SIGHUP, reloading config...")
					prefs, err = MergePrefsFromFile(tailscalePlan.TargetPrefs, configFile)
					tailscalePlan.TargetPrefs = prefs
				}
			case e := <-configChan:
				if e != nil {
					fmt.Printf("error occurred watching file: %v", e)
				}
				fmt.Println("config changed, reloading...")
				prefs, err = MergePrefsFromFile(tailscalePlan.TargetPrefs, configFile)
				tailscalePlan.TargetPrefs = prefs
			}
			ctx, done := context.WithTimeout(ctx, 10*time.Second)
			defer done()
			scheduler.Execute(ctx, tailscalePlan)
		}
	}()

	wg.Wait()
}

func MergePrefsFromFile(prefs *ipn.Prefs, filename string) (*ipn.Prefs, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(b, prefs)
	if err != nil {
		return nil, err
	}
	return prefs, nil
}

type TailscalePlan struct {
	TargetPrefs  *ipn.Prefs
	currentPrefs *ipn.Prefs
	maskedPrefs  *ipn.MaskedPrefs
}

func (t *TailscalePlan) Create(ctx context.Context) (procedure []planner.Procedure, err error) {
	t.currentPrefs, err = tailscale.GetPrefs(ctx)
	if err != nil {
		return
	}
	mask, diffDetected, err := CalculateMaskedPrefs(t.currentPrefs, t.TargetPrefs)
	if err != nil {
		return
	}
	if diffDetected {
		t.maskedPrefs = mask
		return []planner.Procedure{&UpdatePreferences{plan: t}}, nil
	}
	return
}

func (t *TailscalePlan) Name() string {
	return "tailscale_preferences_plan"
}

type UpdatePreferences struct {
	plan *TailscalePlan
}

func (u *UpdatePreferences) Name() string {
	return "update_prefs"
}

func (u *UpdatePreferences) Do(ctx context.Context) (procedure []planner.Procedure, err error) {

	returnedPrefs, err := tailscale.EditPrefs(ctx, u.plan.maskedPrefs)
	if err != nil {
		return
	}
	fetchedPrefs, err := tailscale.GetPrefs(ctx)
	if err != nil {
		return
	}
	if _, diff, err := CalculateMaskedPrefs(returnedPrefs, fetchedPrefs); diff {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("desired preferences failed to apply correctly")
	}
	if _, diff, err := CalculateMaskedPrefs(fetchedPrefs, u.plan.TargetPrefs); diff {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("desired preferences still differ after apply")
	}
	u.plan.currentPrefs = fetchedPrefs
	return
}

func CalculateMaskedPrefs(c, t *ipn.Prefs) (mask *ipn.MaskedPrefs, diff bool, err error) {
	mask = &ipn.MaskedPrefs{
		Prefs: *c,
	}
	if c.ControlURL != t.ControlURL {
		mask.ControlURLSet = true
		diff = true
		mask.Prefs.ControlURL = t.ControlURL
	}
	if c.RouteAll != t.RouteAll {
		mask.RouteAllSet = true
		diff = true
		mask.Prefs.RouteAll = t.RouteAll
	}
	if c.AllowSingleHosts != t.AllowSingleHosts {
		mask.AllowSingleHostsSet = true
		diff = true
		mask.Prefs.AllowSingleHosts = t.AllowSingleHosts
	}
	if c.ExitNodeID != t.ExitNodeID {
		mask.ExitNodeIDSet = true
		diff = true
		mask.Prefs.ExitNodeID = t.ExitNodeID
	}
	if c.ExitNodeIP != t.ExitNodeIP {
		mask.ExitNodeIPSet = true
		diff = true
		mask.Prefs.ExitNodeIP = t.ExitNodeIP
	}
	if c.ExitNodeAllowLANAccess != t.ExitNodeAllowLANAccess {
		mask.ExitNodeAllowLANAccessSet = true
		diff = true
		mask.Prefs.ExitNodeAllowLANAccess = t.ExitNodeAllowLANAccess
	}
	if c.CorpDNS != t.CorpDNS {
		mask.CorpDNSSet = true
		diff = true
		mask.Prefs.CorpDNS = t.CorpDNS
	}
	if c.RunSSH != t.RunSSH {
		mask.RunSSHSet = true
		diff = true
		mask.Prefs.RunSSH = t.RunSSH
	}
	if c.WantRunning != t.WantRunning {
		mask.WantRunningSet = true
		diff = true
		mask.Prefs.WantRunning = t.WantRunning
	}
	if c.LoggedOut != t.LoggedOut {
		mask.LoggedOutSet = true
		diff = true
		mask.Prefs.LoggedOut = t.LoggedOut
	}
	if c.ShieldsUp != t.ShieldsUp {
		mask.ShieldsUpSet = true
		diff = true
		mask.Prefs.ShieldsUp = t.ShieldsUp
	}
	if !reflect.DeepEqual(c.AdvertiseTags, t.AdvertiseTags) {
		mask.AdvertiseTagsSet = true
		diff = true
		for _, tag := range t.AdvertiseTags {
			mask.Prefs.AdvertiseTags = append(mask.Prefs.AdvertiseTags, tag)
		}
	}
	if c.Hostname != t.Hostname {
		mask.HostnameSet = true
		diff = true
		mask.Prefs.Hostname = t.Hostname
	}
	if c.NotepadURLs != t.NotepadURLs {
		mask.NotepadURLsSet = true
		diff = true
		mask.Prefs.NotepadURLs = t.NotepadURLs
	}
	if c.ForceDaemon != t.ForceDaemon {
		mask.ForceDaemonSet = true
		diff = true
		mask.Prefs.ForceDaemon = t.ForceDaemon
	}
	if !reflect.DeepEqual(c.AdvertiseRoutes, t.AdvertiseRoutes) {
		mask.AdvertiseRoutesSet = true
		diff = true
		for _, route := range t.AdvertiseRoutes {
			mask.Prefs.AdvertiseRoutes = append(mask.Prefs.AdvertiseRoutes, route)
		}
	}
	if c.NoSNAT != t.NoSNAT {
		mask.NoSNATSet = true
		diff = true
		mask.Prefs.NoSNAT = t.NoSNAT
	}
	if c.NetfilterMode != t.NetfilterMode {
		mask.NetfilterModeSet = true
		diff = true
		mask.Prefs.NetfilterMode = t.NetfilterMode
	}
	if c.OperatorUser != t.OperatorUser {
		mask.OperatorUserSet = true
		diff = true
		mask.Prefs.OperatorUser = t.OperatorUser
	}
	return
}

func initLogger() *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "console"
	l, _ := cfg.Build()
	return l
}
