package main

import (
	"context"
	"fmt"
	"github.com/ghodss/yaml"
	"tailscale.com/client/tailscale"
)

func main() {
	ctx := context.Background()
	prefs, _ := tailscale.GetPrefs(ctx)
	y, _ := yaml.Marshal(prefs)
	fmt.Println(string(y))
}
