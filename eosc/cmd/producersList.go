// Copyright © 2018 EOS Canada <info@eoscanada.com>

package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/msig"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

		out, err := requestProducers(ctx, api)
		errorCheck("recursing to get producers accounts", err)

		for el := range out {
			chunks := strings.Split(el, "@")
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

		pushEOSCActions(ctx, api,
			msig.NewPropose(proposer, proposalName, requested, tx),
		)
	},
}

func getProducersTable(ctx context.Context, api *eos.API) (prods producers, err error) {
	lowerBound := ""
	for {
		response, err := api.GetTableRows(
			ctx,
			eos.GetTableRowsRequest{
				Scope:      "eosio",
				Code:       "eosio",
				Table:      "producers",
				JSON:       true,
				LowerBound: lowerBound,
				Limit:      5000,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("get producers table: %w", err)
		}

		var rows producers
		json.Unmarshal(response.Rows, &rows)
		if err != nil {
			return nil, fmt.Errorf("json unmarshal: %w", err)
		}

		prods = append(prods, rows...)

		if !response.More {
			break
		}

		if len(rows) != 0 {
			last := rows[len(rows)-1]
			owner := last["owner"].(string)
			val, _ := eos.StringToName(owner)
			lowerBound = eos.NameToString(val + 1)
		}
	}
	return
}

func requestProducers(ctx context.Context, api *eos.API) (out map[string]bool, err error) {
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

func recurseAccounts(ctx context.Context, api *eos.API, in map[string]bool, account string, permission string, level, maxLevels int) (out map[string]bool, err error) {
	out = in

	newAcct := fmt.Sprintf("%s@%s", account, permission)
	if _, found := out[newAcct]; found {
		return
	}

	fmt.Println("      - ADDING:", newAcct)
	out[newAcct] = true

	if level >= maxLevels {
		return out, nil
	}

	//fmt.Println("Fetching account", account)
	resp, err := api.GetAccount(ctx, eos.AccountName(account))
	if err != nil {
		return nil, err
	}

	curPerm := permissionByName(resp.Permissions, permission)
	for {
		if !viper.GetBool("multisig-propose-cmd-with-owner") && curPerm.PermName == "owner" {
			break
		}

		if curPerm.PermName == "" {
			break
		}

		for _, acct := range curPerm.RequiredAuth.Accounts {
			out, err = recurseAccounts(ctx, api, out, string(acct.Permission.Actor), string(acct.Permission.Permission), level+1, maxLevels)
			if err != nil {
				return nil, err
			}
		}

		curPerm = permissionByName(resp.Permissions, curPerm.Parent)
	}

	return
}

func permissionByName(perms []eos.Permission, name string) eos.Permission {
	for _, perm := range perms {
		if perm.PermName == name {
			return perm
		}
	}
	return eos.Permission{}
}

func init() {
	msigCmd.AddCommand(producersListCmd)

	producersListCmd.Flags().StringSlice("request", []string{}, "List producer account sorted by total_votes")
}
