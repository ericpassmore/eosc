// Copyright © 2018 EOS Canada <info@eoscanada.com>

package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	eos "github.com/eoscanada/eos-go"
	"github.com/spf13/cobra"
)

// producersListCmd represents the msigPropose command
var producersListCmd = &cobra.Command{
	Use:   "producersList",
	Short: "See the producers",
	Long: `Get a list of producers sorted by total votes
  `,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		api := getAPI()

    var requested []eos.PermissionLevel
		out, err := requestProducers2(ctx, api)
		errorCheck("recursing to get producers accounts", err)

		for el := range out {
			chunks := strings.Split(el, "@")
      fmt.Printf("Last Loop: %s\n", chunks[0])
			requested = append(requested, eos.PermissionLevel{
				Actor:      eos.AccountName(chunks[0]),
				Permission: eos.PermissionName(chunks[1]),
			})
		}

		sort.Slice(requested, func(i, j int) bool {
			el1 := requested[i]
			el2 := requested[j]
			if el1.Actor < el2.Actor {
				return true
			}
			if el1.Actor > el2.Actor {
				return false
			}
			return el1.Permission < el2.Permission
		})
	},
}

func requestProducers2(ctx context.Context, api *eos.API) (out map[string]bool, err error) {
	producers, err := getProducersTable(ctx, api)
	errorCheck("get producers table", err)

	sort.Slice(producers, producers.Less)

	out = make(map[string]bool)

	for idx, p := range producers {
		if len(out) > 39 {
			break
		}


		fmt.Printf("total votes %s", p["total_votes"])
		newAcct := fmt.Sprintf("%s@active", p["owner"].(string))

		if isActive, _ := p["is_active"].(float64); isActive != 1 {
			fmt.Printf("Skipping inactive no. %d: %s\n", idx+1, newAcct)
			continue
		}

		fmt.Printf("Adding no. %d: %s\n", idx+1, newAcct)
		out[newAcct] = true
	}

	return
}

func init() {
	msigCmd.AddCommand(producersListCmd)

	producersListCmd.Flags().StringSlice("request", []string{}, "List producer account sorted by total_votes")
}
